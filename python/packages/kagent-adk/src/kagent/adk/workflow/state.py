"""Workflow state management for KAgent workflow agents.

This module provides data models and state management for workflow agents
that use outputKey to pass data between sub-agents with separate sessions.
"""

import asyncio
import logging
import time
from datetime import datetime, timezone
from enum import Enum
from typing import Dict, List, Optional

from pydantic import BaseModel, Field

logger = logging.getLogger(__name__)

# Prometheus metrics for parallel workflow monitoring
try:
    from prometheus_client import Counter, Histogram

    # Counter for total concurrent writes
    parallel_state_writes_total = Counter(
        "kagent_parallel_workflow_state_writes_total",
        "Total number of concurrent state writes to workflow state",
        ["workflow_name", "namespace"],
    )

    # Histogram for lock wait times
    parallel_lock_wait_seconds = Histogram(
        "kagent_parallel_workflow_lock_wait_seconds",
        "Time spent waiting for workflow state lock",
        ["workflow_name", "namespace"],
        buckets=[0.000001, 0.00001, 0.0001, 0.001, 0.01, 0.1],  # 1μs to 100ms
    )

    METRICS_AVAILABLE = True
except ImportError:
    logger.warning("prometheus_client not available - metrics disabled")
    METRICS_AVAILABLE = False


class WorkflowStatus(str, Enum):
    """Workflow execution status."""

    RUNNING = "running"
    COMPLETED = "completed"
    FAILED = "failed"
    CANCELLED = "cancelled"


class SubAgentExecution(BaseModel):
    """Record of a single sub-agent execution within a workflow."""

    index: int = Field(..., description="Execution order (0-indexed)")
    agent_name: str = Field(..., description="Name of the sub-agent")
    agent_namespace: str = Field(default="default", description="Namespace of the sub-agent")
    session_id: str = Field(..., description="Session ID for this execution")
    output_key: Optional[str] = Field(None, description="OutputKey where output was stored")

    started_at: datetime = Field(..., description="Execution start time")
    completed_at: Optional[datetime] = Field(None, description="Execution completion time")

    status: str = Field(..., description="Execution status (success/failed/cancelled)")
    output_size_bytes: int = Field(default=0, description="Size of output value in bytes")
    error: Optional[str] = Field(None, description="Error message if failed")

    # NEW: Completion order for parallel workflows
    completion_order: Optional[int] = Field(
        None,
        description="Completion order for parallel agents (1=first, 2=second, etc.). None for sequential workflows.",
    )


class WorkflowState(BaseModel):
    """Complete state of a workflow execution.

    This model represents the persistent state of a workflow, including:
    - State data (outputKey → value mappings)
    - Sub-agent execution history
    - Workflow status and metadata
    """

    # Identification
    workflow_session_id: str = Field(..., description="Unique workflow session ID")
    user_id: str = Field(..., description="User who triggered the workflow")
    agent_name: str = Field(..., description="Name of the workflow agent")
    namespace: str = Field(default="default", description="Kubernetes namespace")

    # State data (outputKey → value mapping)
    state_data: Dict[str, str] = Field(default_factory=dict, description="OutputKey values")

    # Execution history
    sub_agent_executions: List[SubAgentExecution] = Field(
        default_factory=list, description="History of sub-agent executions"
    )

    # Workflow metadata
    status: WorkflowStatus = Field(default=WorkflowStatus.RUNNING, description="Workflow status")
    error_message: Optional[str] = Field(None, description="Error details if failed")

    # Timestamps
    created_at: datetime = Field(default_factory=lambda: datetime.now(timezone.utc), description="Workflow start time")
    updated_at: datetime = Field(
        default_factory=lambda: datetime.now(timezone.utc), description="Last state update time"
    )
    completed_at: Optional[datetime] = Field(None, description="Workflow completion time")

    class Config:
        use_enum_values = True

    def get_output(self, key: str) -> Optional[str]:
        """Get output value by key.

        Args:
            key: OutputKey name

        Returns:
            Output value or None if not found
        """
        return self.state_data.get(key)

    def set_output(self, key: str, value: str, max_size_bytes: int = 10 * 1024 * 1024) -> None:
        """Set output value for a key.

        Args:
            key: OutputKey name
            value: Output value to store
            max_size_bytes: Maximum allowed size (default 10MB)

        Raises:
            ValueError: If value exceeds max_size_bytes
        """
        value_bytes = len(value.encode("utf-8"))
        if value_bytes > max_size_bytes:
            raise ValueError(
                f"Output value for key '{key}' exceeds maximum size ({value_bytes} > {max_size_bytes} bytes)"
            )
        self.state_data[key] = value
        self.updated_at = datetime.now(timezone.utc)

    def add_execution(self, execution: SubAgentExecution) -> None:
        """Add a sub-agent execution record.

        Args:
            execution: SubAgentExecution record to add
        """
        self.sub_agent_executions.append(execution)
        self.updated_at = datetime.now(timezone.utc)

    def mark_completed(self) -> None:
        """Mark workflow as completed."""
        self.status = WorkflowStatus.COMPLETED
        self.completed_at = datetime.now(timezone.utc)
        self.updated_at = datetime.now(timezone.utc)

    def mark_failed(self, error_message: str) -> None:
        """Mark workflow as failed.

        Args:
            error_message: Error description
        """
        self.status = WorkflowStatus.FAILED
        self.error_message = error_message
        self.completed_at = datetime.now(timezone.utc)
        self.updated_at = datetime.now(timezone.utc)


class WorkflowStateManager:
    """Manager for workflow state with in-memory caching and database persistence.

    This class provides methods to create, retrieve, and update workflow state,
    with an in-memory cache for performance and async database persistence.

    For parallel workflows, this manager provides thread-safe concurrent write methods
    using asyncio.Lock to prevent race conditions when multiple agents complete simultaneously.
    """

    def __init__(self):
        """Initialize the workflow state manager."""
        self._cache: Dict[str, WorkflowState] = {}
        self._lock = asyncio.Lock()  # Thread-safe concurrent writes for parallel workflows

    def create_workflow(
        self, workflow_session_id: str, user_id: str, agent_name: str, namespace: str = "default"
    ) -> WorkflowState:
        """Create a new workflow state.

        Args:
            workflow_session_id: Unique workflow session ID
            user_id: User who triggered the workflow
            agent_name: Name of the workflow agent
            namespace: Kubernetes namespace

        Returns:
            Newly created WorkflowState
        """
        workflow_state = WorkflowState(
            workflow_session_id=workflow_session_id,
            user_id=user_id,
            agent_name=agent_name,
            namespace=namespace,
        )
        self._cache[workflow_session_id] = workflow_state
        return workflow_state

    def get_workflow(self, workflow_session_id: str) -> Optional[WorkflowState]:
        """Get workflow state from cache or database.

        Args:
            workflow_session_id: Workflow session ID to retrieve

        Returns:
            WorkflowState if found, None otherwise
        """
        return self._cache.get(workflow_session_id)

    def update_output(self, workflow_session_id: str, output_key: str, output_value: str) -> None:
        """Update output value in workflow state.

        Args:
            workflow_session_id: Workflow session ID
            output_key: OutputKey name
            output_value: Output value to store

        Raises:
            ValueError: If workflow not found or value too large
        """
        workflow_state = self.get_workflow(workflow_session_id)
        if workflow_state is None:
            raise ValueError(f"Workflow with session ID '{workflow_session_id}' not found")
        workflow_state.set_output(output_key, output_value)

    def get_outputs(self, workflow_session_id: str) -> Dict[str, str]:
        """Get all output values from workflow state.

        Args:
            workflow_session_id: Workflow session ID

        Returns:
            Dictionary of outputKey → value mappings

        Raises:
            ValueError: If workflow not found
        """
        workflow_state = self.get_workflow(workflow_session_id)
        if workflow_state is None:
            raise ValueError(f"Workflow with session ID '{workflow_session_id}' not found")
        return workflow_state.state_data.copy()

    async def update_output_concurrent(self, workflow_session_id: str, key: str, value: str) -> None:
        """Update workflow state output with thread-safe locking for parallel workflows.

        This method MUST be used by parallel workflows to prevent race conditions
        when multiple agents complete simultaneously. The asyncio.Lock ensures
        atomic writes to the workflow state.

        Metrics:
            - Tracks lock wait time (time spent waiting to acquire lock)
            - Increments concurrent write counter

        Args:
            workflow_session_id: Workflow session ID
            key: OutputKey name
            value: Output value to store

        Raises:
            KeyError: If workflow not found
            ValueError: If value exceeds size limits
        """
        # Measure lock wait time
        lock_start_time = time.perf_counter()

        async with self._lock:
            lock_wait_time = time.perf_counter() - lock_start_time

            # Atomic write to workflow state
            workflow_state = self._cache.get(workflow_session_id)
            if workflow_state is None:
                raise KeyError(f"Workflow with session ID '{workflow_session_id}' not found")

            # Log lock acquisition and wait time
            logger.debug(
                f"Acquired lock for concurrent output write: "
                f"workflow_session_id={workflow_session_id}, key={key}, "
                f"lock_wait_time={lock_wait_time * 1e6:.2f}μs"
            )

            # Perform atomic write
            workflow_state.set_output(key, value)

            # Update metrics
            if METRICS_AVAILABLE:
                parallel_state_writes_total.labels(
                    workflow_name=workflow_state.agent_name, namespace=workflow_state.namespace
                ).inc()

                parallel_lock_wait_seconds.labels(
                    workflow_name=workflow_state.agent_name, namespace=workflow_state.namespace
                ).observe(lock_wait_time)

            logger.debug(f"Completed concurrent output write: workflow_session_id={workflow_session_id}, key={key}")

    async def add_execution_concurrent(self, workflow_session_id: str, execution: SubAgentExecution) -> None:
        """Add execution record with thread-safe locking for parallel workflows.

        This method ensures atomic updates to the sub_agent_executions list
        when multiple parallel agents complete simultaneously.

        Args:
            workflow_session_id: Workflow session ID
            execution: SubAgentExecution record to add

        Raises:
            KeyError: If workflow not found
        """
        async with self._lock:
            workflow_state = self._cache.get(workflow_session_id)
            if workflow_state is None:
                raise KeyError(f"Workflow with session ID '{workflow_session_id}' not found")

            workflow_state.add_execution(execution)

            logger.debug(
                f"Added execution record: workflow_session_id={workflow_session_id}, "
                f"agent={execution.agent_name}, completion_order={execution.completion_order}"
            )
