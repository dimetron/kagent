"""KAgent ParallelAgent with max_workers concurrency limiting and outputKey support."""

import asyncio
import logging
import time
from datetime import datetime, timezone
from typing import AsyncGenerator

from google.adk.agents import ParallelAgent
from google.adk.agents.base_agent import BaseAgent
from google.adk.agents.invocation_context import InvocationContext, Session
from google.adk.events import Event
from google.genai.types import Content, Part
from opentelemetry import trace

from ..metrics import ParallelAgentMetrics, set_max_workers_metric
from ..workflow.state import SubAgentExecution, WorkflowStateManager

logger = logging.getLogger(__name__)
tracer = trace.get_tracer(__name__)


class KAgentParallelAgent(ParallelAgent):
    """ParallelAgent with configurable max_workers concurrency limiting.

    Extends Google ADK's ParallelAgent to add semaphore-based concurrency control,
    limiting the number of sub-agents executing concurrently to prevent resource
    exhaustion.

    Attributes:
        max_workers: Maximum number of sub-agents executing concurrently (default: 10)
        semaphore: Asyncio semaphore for concurrency control (initialized in __init__)
    """

    # Declare max_workers as a Pydantic field
    max_workers: int = 10
    namespace: str = "default"

    # Configure model to allow extra fields for semaphore
    model_config = {"extra": "allow", "arbitrary_types_allowed": True}

    def __init__(
        self,
        *,
        name: str,
        description: str = "",
        parent_agent: BaseAgent | None = None,
        sub_agents: list[BaseAgent] | None = None,
        max_workers: int = 5,
        namespace: str = "default",
        **kwargs,
    ):
        """Initialize KAgentParallelAgent with max_workers support.

        Args:
            name: Agent name
            description: Agent description
            parent_agent: Parent agent (optional)
            sub_agents: List of sub-agents to execute in parallel
            max_workers: Maximum concurrent sub-agents (default: 5, min: 1, max: 50)
            namespace: Kubernetes namespace for metrics (default: "default")
            **kwargs: Additional arguments passed to ParallelAgent
        """
        # Validate max_workers
        if not isinstance(max_workers, int):
            raise TypeError(f"max_workers must be int, got {type(max_workers)}")
        if max_workers < 1:
            raise ValueError(f"max_workers must be >= 1, got {max_workers}")
        if max_workers > 50:
            raise ValueError(f"max_workers must be <= 50, got {max_workers}")

        # Initialize parent with all fields including max_workers
        super().__init__(
            name=name,
            description=description,
            parent_agent=parent_agent,
            sub_agents=sub_agents or [],
            max_workers=max_workers,
            namespace=namespace,
            **kwargs,
        )

        # Create semaphore for concurrency control
        self.semaphore = asyncio.Semaphore(max_workers)

        # Initialize Prometheus metrics
        set_max_workers_metric(name, self.namespace, max_workers)

        logger.info(
            f"Initialized KAgentParallelAgent '{name}' with max_workers={max_workers}, "
            f"sub_agents={len(self.sub_agents)}"
        )

    async def run_async(self, parent_context: InvocationContext) -> AsyncGenerator[Event, None]:
        """Execute sub-agents in parallel with concurrency limiting and optional outputKey support.

        This method supports two modes:

        1. **outputKey mode** (if any sub-agent has output_key): Creates separate sessions for each
           sub-agent, collects outputs in workflow state, and injects state into subsequent agents.
           **This is the recommended mode for aggregation workflows**.

        2. **Shared session mode** (default): All sub-agents share the parent session (existing behavior).

        **OutputKey Mode - Aggregation Pattern:**

        When you define `output_key` on one or more sub-agents, the parallel agent automatically:

        1. **Creates Separate Sessions**: Each sub-agent gets its own session ID to isolate execution
        2. **Collects Outputs**: Captures all text output from each sub-agent's events
        3. **Stores in Workflow State**: Uses thread-safe concurrent writes to store outputs by key
        4. **Injects into Parent Session**: After parallel execution completes, injects all outputs
           into `parent_context.session.state`, making them accessible to subsequent agents

        **Example - Parallel Data Collection → Aggregation:**

        ```python
        # Define sub-agents with outputKey
        east_collector = RemoteA2aAgent(name="east_collector", output_key="east_data")
        west_collector = RemoteA2aAgent(name="west_collector", output_key="west_data")
        central_collector = RemoteA2aAgent(name="central_collector", output_key="central_data")

        # Create parallel workflow
        parallel_workflow = KAgentParallelAgent(
            name="data_collection_workflow",
            sub_agents=[east_collector, west_collector, central_collector],
            max_workers=3,
        )

        # After parallel execution completes, session.state contains:
        # {
        #   "east_data": "... output from east collector ...",
        #   "west_data": "... output from west collector ...",
        #   "central_data": "... output from central collector ..."
        # }

        # Aggregator agent can access all outputs via session.state
        aggregator = RemoteA2aAgent(
            name="aggregator", system_message="Aggregate data from east_data, west_data, central_data in session.state"
        )
        ```

        **Thread Safety:**

        Parallel workflows use `asyncio.Lock` to ensure thread-safe concurrent writes to workflow state.
        This prevents data loss or corruption when multiple agents complete simultaneously.

        **Completion Order Tracking:**

        Each sub-agent execution is tracked with a `completion_order` field (1=first, 2=second, etc.)
        to help with debugging and performance analysis.

        **Hybrid Workflows:**

        State injection supports complex workflows like:
        - Sequential validation → Parallel deployment → Sequential reporting
        - Parallel data collection → Sequential analysis → Parallel processing

        The injected state persists across all subsequent agents in the workflow chain.

        Args:
            parent_context: Invocation context from parent agent

        Yields:
            Event: Events generated by sub-agents during execution

        See Also:
            - WorkflowStateManager: Manages thread-safe state updates
            - inject_state_keys(): Used for sequential workflows with outputKey
            - KAgentSequentialAgent: For sequential aggregation patterns
        """
        # T011: Detect outputKey mode
        use_output_key_mode = any(hasattr(agent, "output_key") and agent.output_key for agent in self.sub_agents)

        # Initialize workflow state manager if using outputKey mode
        workflow_state_manager = None
        workflow_state = None

        if use_output_key_mode:
            # T012: Create workflow state
            workflow_state_manager = WorkflowStateManager()
            workflow_state = workflow_state_manager.create_workflow(
                workflow_session_id=parent_context.session.id,
                user_id=parent_context.user_id or parent_context.session.user_id,
                agent_name=self.name,
                namespace=self.namespace,
            )
            logger.info(
                f"Parallel workflow '{self.name}' using outputKey mode with "
                f"{len([a for a in self.sub_agents if hasattr(a, 'output_key') and a.output_key])} agents "
                f"with outputKey"
            )

        with tracer.start_as_current_span(
            f"{self.name}.run_async",
            attributes={
                "kagent.agent.name": self.name,
                "kagent.agent.type": "parallel",
                "kagent.parallel.max_workers": self.max_workers,
                "kagent.parallel.sub_agent_count": len(self.sub_agents),
                "kagent.parallel.output_key_mode": use_output_key_mode,
            },
        ) as span:
            logger.info(
                f"Starting KAgentParallelAgent '{self.name}' with "
                f"{len(self.sub_agents)} sub-agents, max_workers={self.max_workers}, "
                f"outputKey_mode={use_output_key_mode}"
            )

            # T015: Create tasks for all sub-agents (adapted for outputKey mode)
            tasks = []
            for idx, sub_agent in enumerate(self.sub_agents):
                if use_output_key_mode:
                    # T014: Use output collection helper for outputKey mode
                    task = asyncio.create_task(
                        self._execute_sub_agent_with_output_collection(
                            sub_agent, idx, parent_context, workflow_state, workflow_state_manager
                        )
                    )
                else:
                    # Existing behavior: shared session mode
                    task = asyncio.create_task(self._run_sub_agent_with_semaphore(sub_agent, parent_context, idx))
                tasks.append(task)

            # Wait for all tasks to complete and collect results
            results = await asyncio.gather(*tasks, return_exceptions=True)

            # T016: Handle partial success - process results
            completed_count = 0
            error_count = 0

            for idx, result in enumerate(results):
                if isinstance(result, Exception):
                    error_count += 1
                    sub_agent_name = self.sub_agents[idx].name
                    error_msg = str(result)
                    logger.error(
                        f"Sub-agent {sub_agent_name} failed: {error_msg}",
                        exc_info=result,
                    )
                    # Yield error event with proper structure
                    yield Event(
                        author=self.name,
                        content=Content(
                            parts=[Part(text=f"Error in sub-agent {sub_agent_name}: {error_msg}")],
                            role="model",
                        ),
                        error_code="SUB_AGENT_ERROR",
                        error_message=error_msg,
                        custom_metadata={"sub_agent": sub_agent_name, "error": True},
                    )
                else:
                    completed_count += 1
                    # Yield events from successful execution
                    for event in result:
                        yield event

            # Mark workflow state as completed or failed
            if use_output_key_mode and workflow_state:
                if completed_count > 0:
                    workflow_state.mark_completed()
                    logger.info(
                        f"Parallel workflow '{self.name}' completed: {completed_count} successful, "
                        f"{error_count} failed, {len(workflow_state.state_data)} outputs stored"
                    )
                else:
                    workflow_state.mark_failed("All parallel agents failed")
                    logger.error(f"Parallel workflow '{self.name}' failed: all {len(self.sub_agents)} agents failed")

                # T020: Inject workflow state into parent session for next phase (aggregators)
                # This makes parallel outputs accessible to subsequent agents via session.state
                if workflow_state.state_data:
                    # Merge workflow state data into parent session state
                    # This allows aggregators to access parallel outputs via session.state
                    parent_context.session.state.update(workflow_state.state_data)

                    logger.info(
                        f"Injected {len(workflow_state.state_data)} workflow outputs into session state "
                        f"(keys: {', '.join(workflow_state.state_data.keys())})"
                    )

            # T017: Set OpenTelemetry attributes
            span.set_attribute("kagent.parallel.completed_count", completed_count)
            span.set_attribute("kagent.parallel.error_count", error_count)
            if use_output_key_mode and workflow_state:
                span.set_attribute("kagent.parallel.output_keys_collected", len(workflow_state.state_data))
                span.set_attribute("kagent.parallel.state_keys_injected", len(workflow_state.state_data))

            logger.info(
                f"KAgentParallelAgent '{self.name}' completed: {completed_count} successful, {error_count} errors"
            )

    async def _execute_sub_agent_with_output_collection(
        self, sub_agent: BaseAgent, idx: int, parent_context: InvocationContext, workflow_state, workflow_state_manager
    ) -> list[Event]:
        """Execute sub-agent with output collection and workflow state management (T014).

        This method:
        1. Creates a separate session ID for the sub-agent (T013)
        2. Executes the sub-agent with semaphore control
        3. Collects output events
        4. Writes outputs to workflow state using thread-safe concurrent methods
        5. Tracks completion order
        6. Adds execution records

        Args:
            sub_agent: Sub-agent to execute
            idx: Index of sub-agent in sub_agents list
            parent_context: Parent invocation context
            workflow_state: WorkflowState for this workflow
            workflow_state_manager: WorkflowStateManager for concurrent writes

        Returns:
            List of events generated by the sub-agent
        """
        # T013: Create separate session ID for this sub-agent
        sub_agent_session_id = f"{parent_context.session.id}-sub-{idx}"

        # Get output_key if defined
        output_key = getattr(sub_agent, "output_key", None)

        # Create metrics context manager
        metrics = ParallelAgentMetrics(agent_name=self.name, namespace=self.namespace)

        # Increment queue depth before waiting for semaphore
        metrics.inc_queue_depth()
        semaphore_wait_start = time.time()

        # Acquire semaphore (blocks if max_workers limit reached)
        async with self.semaphore:
            # Decrement queue depth once semaphore acquired
            metrics.dec_queue_depth()
            semaphore_wait_ms = (time.time() - semaphore_wait_start) * 1000

            # Determine if output_key was auto-generated (sub_agent has output_key attribute but it's being set by workflow)
            output_key_source = (
                "automatic"
                if not hasattr(sub_agent, "_original_output_key")
                or sub_agent.output_key != getattr(sub_agent, "_original_output_key", None)
                else "manual"
            )
            # Check if output_key was auto-generated by looking at the sub_agent's original definition
            # If sub_agent.output_key exists, it means it was set (either explicitly or auto-generated)
            # We consider it auto-generated if the format matches {namespace}_{name} pattern
            is_auto_generated = False
            if output_key and hasattr(sub_agent, "output_key") and sub_agent.output_key:
                # Simple heuristic: if output_key contains underscore and matches typical auto-gen pattern, it's likely auto-generated
                # A more robust check would require storing the original SubAgentReference, but this is sufficient
                is_auto_generated = "_" in output_key and not any(c in output_key for c in ["-", " ", "."])

            with tracer.start_as_current_span(
                f"{self.name}.sub_agent[{idx}].{sub_agent.name}",
                attributes={
                    "kagent.agent.name": sub_agent.name,
                    "kagent.parallel.sub_agent_index": idx,
                    "kagent.parallel.semaphore_wait_ms": semaphore_wait_ms,
                    "kagent.parallel.max_workers": self.max_workers,
                    "kagent.parallel.output_key": output_key or "none",
                    "kagent.workflow.output_key_auto_generated": is_auto_generated,
                    "kagent.workflow.output_key_source": output_key_source,
                },
            ) as span:
                if semaphore_wait_ms > 10:  # Log if waited more than 10ms
                    logger.debug(
                        f"Sub-agent {sub_agent.name} waited {semaphore_wait_ms:.1f}ms "
                        f"for semaphore (max_workers={self.max_workers})"
                    )

                execution_start = time.time()
                started_at = datetime.now(timezone.utc)

                try:
                    # Track active execution with metrics
                    async with metrics:
                        # T030: Create separate session for this sub-agent with state injection
                        # Merge parent session state (from previous phases) with workflow state (current workflow)
                        # This enables hybrid workflows: sequential → parallel → sequential
                        merged_state = {}
                        if parent_context.session.state:
                            merged_state.update(parent_context.session.state)
                        if workflow_state and workflow_state.state_data:
                            merged_state.update(workflow_state.state_data)

                        sub_agent_session = Session(
                            id=sub_agent_session_id,
                            user_id=parent_context.session.user_id,
                            app_name=parent_context.session.app_name,
                            state=merged_state,
                        )

                        # Create sub-agent context with separate session
                        # Pass all required fields from parent context to comply with Google ADK InvocationContext API
                        sub_context = InvocationContext(
                            session=sub_agent_session,
                            session_service=parent_context.session_service,
                            invocation_id=parent_context.invocation_id,
                            agent=parent_context.agent,
                        )

                        # Execute sub-agent and collect events
                        events = []
                        collected_output = []

                        async for event in sub_agent.run_async(sub_context):
                            events.append(event)

                            # Collect output text if this sub-agent has an output_key
                            if output_key and event.content:
                                # Extract text from event content
                                if hasattr(event.content, "parts") and event.content.parts:
                                    for part in event.content.parts:
                                        if hasattr(part, "text") and part.text:
                                            collected_output.append(part.text)

                    execution_ms = (time.time() - execution_start) * 1000
                    completed_at = datetime.now(timezone.utc)
                    span.set_attribute("kagent.agent.execution_ms", execution_ms)
                    span.set_attribute("kagent.agent.event_count", len(events))

                    # Write output to workflow state if output_key is defined
                    output_value = ""
                    if output_key and collected_output:
                        output_value = "\n".join(collected_output)

                        # Use thread-safe concurrent write
                        await workflow_state_manager.update_output_concurrent(
                            workflow_session_id=workflow_state.workflow_session_id, key=output_key, value=output_value
                        )

                        logger.debug(
                            f"Stored output for {sub_agent.name}: key={output_key}, size={len(output_value)} bytes"
                        )

                    # Track completion order and add execution record
                    # Use lock to atomically increment completion counter
                    async with workflow_state_manager._lock:
                        completion_order = len(workflow_state.sub_agent_executions) + 1

                        execution_record = SubAgentExecution(
                            index=idx,
                            agent_name=sub_agent.name,
                            agent_namespace=self.namespace,
                            session_id=sub_agent_session_id,
                            output_key=output_key,
                            started_at=started_at,
                            completed_at=completed_at,
                            status="success",
                            output_size_bytes=len(output_value.encode("utf-8")) if output_value else 0,
                            completion_order=completion_order,
                        )

                        workflow_state.add_execution(execution_record)

                    logger.debug(
                        f"Sub-agent {sub_agent.name} completed in {execution_ms:.1f}ms, "
                        f"generated {len(events)} events, completion_order={completion_order}"
                    )

                    return events

                except Exception as e:
                    execution_ms = (time.time() - execution_start) * 1000
                    completed_at = datetime.now(timezone.utc)

                    span.set_attribute("kagent.agent.execution_ms", execution_ms)
                    span.set_attribute("error", True)
                    span.record_exception(e)

                    # Record failed execution
                    async with workflow_state_manager._lock:
                        completion_order = None  # Failed agents don't get completion order

                        execution_record = SubAgentExecution(
                            index=idx,
                            agent_name=sub_agent.name,
                            agent_namespace=self.namespace,
                            session_id=sub_agent_session_id,
                            output_key=output_key,
                            started_at=started_at,
                            completed_at=completed_at,
                            status="failed",
                            output_size_bytes=0,
                            error=str(e),
                            completion_order=completion_order,
                        )

                        workflow_state.add_execution(execution_record)

                    logger.error(
                        f"Sub-agent {sub_agent.name} failed after {execution_ms:.1f}ms: {e}",
                        exc_info=e,
                    )

                    # Re-raise to be caught by gather()
                    raise

    async def _run_sub_agent_with_semaphore(
        self, sub_agent: BaseAgent, parent_context: InvocationContext, idx: int
    ) -> list[Event]:
        """Execute a single sub-agent with semaphore-based concurrency control.

        Args:
            sub_agent: Sub-agent to execute
            parent_context: Parent invocation context
            idx: Index of sub-agent in sub_agents list

        Returns:
            List of events generated by the sub-agent

        Raises:
            Exception: Any exception raised by the sub-agent (propagated to caller)
        """
        # Create metrics context manager
        metrics = ParallelAgentMetrics(agent_name=self.name, namespace=self.namespace)

        # Increment queue depth before waiting for semaphore
        metrics.inc_queue_depth()
        semaphore_wait_start = time.time()

        # Acquire semaphore (blocks if max_workers limit reached)
        async with self.semaphore:
            # Decrement queue depth once semaphore acquired
            metrics.dec_queue_depth()
            semaphore_wait_ms = (time.time() - semaphore_wait_start) * 1000

            with tracer.start_as_current_span(
                f"{self.name}.sub_agent[{idx}].{sub_agent.name}",
                attributes={
                    "kagent.agent.name": sub_agent.name,
                    "kagent.parallel.sub_agent_index": idx,
                    "kagent.parallel.semaphore_wait_ms": semaphore_wait_ms,
                    "kagent.parallel.max_workers": self.max_workers,
                },
            ) as span:
                if semaphore_wait_ms > 10:  # Log if waited more than 10ms
                    logger.debug(
                        f"Sub-agent {sub_agent.name} waited {semaphore_wait_ms:.1f}ms "
                        f"for semaphore (max_workers={self.max_workers})"
                    )

                execution_start = time.time()

                try:
                    # Track active execution with metrics
                    async with metrics:
                        # Clone parent context for isolated execution
                        # Use shallow copy to avoid copying unpicklable objects like locks
                        sub_context = parent_context.model_copy(deep=False)

                        # Execute sub-agent and collect events
                        events = []
                        async for event in sub_agent.run_async(sub_context):
                            events.append(event)

                    execution_ms = (time.time() - execution_start) * 1000
                    span.set_attribute("kagent.agent.execution_ms", execution_ms)
                    span.set_attribute("kagent.agent.event_count", len(events))

                    logger.debug(
                        f"Sub-agent {sub_agent.name} completed in {execution_ms:.1f}ms, generated {len(events)} events"
                    )

                    return events

                except Exception as e:
                    execution_ms = (time.time() - execution_start) * 1000
                    span.set_attribute("kagent.agent.execution_ms", execution_ms)
                    span.set_attribute("error", True)
                    span.record_exception(e)

                    logger.error(
                        f"Sub-agent {sub_agent.name} failed after {execution_ms:.1f}ms: {e}",
                        exc_info=e,
                    )

                    # Re-raise to be caught by gather()
                    raise
