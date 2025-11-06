from __future__ import annotations

import asyncio
import inspect
import logging
import uuid
from datetime import datetime, timezone
from typing import Any, Awaitable, Callable

from a2a.server.agent_execution import AgentExecutor
from a2a.server.agent_execution.context import RequestContext
from a2a.server.events.event_queue import EventQueue
from a2a.types import (
    Artifact,
    Message,
    Part,
    Role,
    TaskArtifactUpdateEvent,
    TaskState,
    TaskStatus,
    TaskStatusUpdateEvent,
    TextPart,
)
from google.adk.events import Event, EventActions
from google.adk.runners import Runner
from google.adk.utils.context_utils import Aclosing
from opentelemetry import trace
from typing_extensions import override

from kagent.core.a2a import TaskResultAggregator, get_kagent_metadata_key

from .converters.event_converter import convert_event_to_a2a_events
from .converters.request_converter import convert_a2a_request_to_adk_run_args

logger = logging.getLogger("kagent_adk." + __name__)

HEARTBEAT_INTERVAL_SECONDS = 15


class A2aAgentExecutor(AgentExecutor):
    """Executes ADK agents in response to A2A requests.
    
    This executor:
    - Accepts a callable that returns a Runner instance
    - Converts A2A requests to ADK format
    - Streams events back to the A2A event queue
    - Sends periodic heartbeats for long-running operations
    """

    def __init__(self, *, runner: Callable[..., Runner | Awaitable[Runner]]):
        super().__init__()
        self._runner = runner

    async def _resolve_runner(self) -> Runner:
        """Resolve the runner callable to a Runner instance."""
        if not callable(self._runner):
            raise TypeError(f"Runner must be callable, got {type(self._runner)}")

        result = self._runner()
        runner = await result if inspect.iscoroutine(result) else result

        if not isinstance(runner, Runner):
            raise TypeError(f"Callable must return a Runner instance, got {type(runner)}")

        return runner

    @override
    async def cancel(self, context: RequestContext, event_queue: EventQueue):
        """Cancel the execution."""
        # TODO: Implement proper cancellation logic if needed
        raise NotImplementedError("Cancellation is not supported")

    @override
    async def execute(self, context: RequestContext, event_queue: EventQueue):
        """Execute an A2A request and stream updates to the event queue."""
        if not context.message:
            raise ValueError("A2A request must have a message")

        if not context.current_task:
            await self._publish_status_update(
                event_queue, context, TaskState.submitted, message=context.message
            )

        runner = await self._resolve_runner()
        try:
            await self._handle_request(context, event_queue, runner)
        except Exception as e:
            logger.error("Error handling A2A request: %s", e, exc_info=True)
            await self._publish_failure(event_queue, context, e)

    async def _handle_request(
        self,
        context: RequestContext,
        event_queue: EventQueue,
        runner: Runner,
    ):
        run_args = convert_a2a_request_to_adk_run_args(context)
        session = await self._prepare_session(context, run_args, runner)

        # Store request headers in session state
        await self._update_session_headers(runner, session, context)

        # Set telemetry attributes
        self._set_trace_attributes(context, run_args)

        invocation_context = runner._new_invocation_context(
            session=session,
            new_message=run_args["new_message"],
            run_config=run_args["run_config"],
        )

        # Publish initial working status
        await self._publish_status_update(
            event_queue,
            context,
            TaskState.working,
            metadata=self._create_metadata(runner, run_args),
        )

        task_result_aggregator = TaskResultAggregator()
        heartbeat_task = asyncio.create_task(
            self._send_heartbeat_updates(event_queue, context, runner, run_args)
        )

        try:
            async with Aclosing(runner.run_async(**run_args)) as agen:
                async for adk_event in agen:
                    for a2a_event in convert_event_to_a2a_events(
                        adk_event, invocation_context, context.task_id, context.context_id
                    ):
                        task_result_aggregator.process_event(a2a_event)
                        await event_queue.enqueue_event(a2a_event)
        finally:
            heartbeat_task.cancel()
            try:
                await heartbeat_task
            except asyncio.CancelledError:
                pass

        await self._publish_final_result(event_queue, context, task_result_aggregator)

    async def _prepare_session(self, context: RequestContext, run_args: dict[str, Any], runner: Runner):
        """Get or create a session for the request."""
        session = await runner.session_service.get_session(
            app_name=runner.app_name,
            user_id=run_args["user_id"],
            session_id=run_args["session_id"],
        )

        if session is None:
            session_name = self._extract_session_name(context.message)
            session = await runner.session_service.create_session(
                app_name=runner.app_name,
                user_id=run_args["user_id"],
                state={"session_name": session_name},
                session_id=run_args["session_id"],
            )
            run_args["session_id"] = session.id

        return session

    def _extract_session_name(self, message: Message | None) -> str | None:
        """Extract session name from the first text part of a message."""
        if not message or not message.parts:
            return None

        for part in message.parts:
            if isinstance(part, Part):
                root_part = part.root
                if isinstance(root_part, TextPart) and root_part.text:
                    text = root_part.text.strip()
                    return text[:20] + ("..." if len(text) > 20 else "")

        return None

    async def _send_heartbeat_updates(
        self,
        event_queue: EventQueue,
        context: RequestContext,
        runner: Runner,
        run_args: dict[str, Any],
    ):
        """Send periodic heartbeat updates for long-running operations."""
        heartbeat_count = 0

        try:
            while True:
                await asyncio.sleep(HEARTBEAT_INTERVAL_SECONDS)
                heartbeat_count += 1

                metadata = self._create_metadata(runner, run_args)
                metadata.update({
                    get_kagent_metadata_key("heartbeat"): "true",
                    get_kagent_metadata_key("heartbeat_count"): str(heartbeat_count),
                    get_kagent_metadata_key("heartbeat_message"): "Agent is working on your request...",
                })

                await self._publish_status_update(
                    event_queue, context, TaskState.working, metadata=metadata
                )

                logger.debug(
                    "Sent heartbeat #%d for task %s (session: %s)",
                    heartbeat_count,
                    context.task_id,
                    run_args["session_id"],
                )

        except asyncio.CancelledError:
            logger.debug(
                "Heartbeat stopped after %d updates for task %s", heartbeat_count, context.task_id
            )
            raise

    async def _update_session_headers(self, runner: Runner, session: Any, context: RequestContext):
        """Store request headers in session state."""
        headers = context.call_context.state.get("headers", {})
        system_event = Event(
            invocation_id="header_update",
            author="system",
            actions=EventActions(state_delta={"headers": headers}),
        )
        await runner.session_service.append_event(session, system_event)

    def _set_trace_attributes(self, context: RequestContext, run_args: dict[str, Any]):
        """Set OpenTelemetry trace attributes."""
        current_span = trace.get_current_span()
        if run_args["user_id"]:
            current_span.set_attribute("kagent.user_id", run_args["user_id"])
        if context.task_id:
            current_span.set_attribute("gen_ai.task.id", context.task_id)
        if run_args["session_id"]:
            current_span.set_attribute("gen_ai.conversation.id", run_args["session_id"])

    def _create_metadata(self, runner: Runner, run_args: dict[str, Any]) -> dict[str, str]:
        """Create metadata dictionary for status updates."""
        return {
            get_kagent_metadata_key("app_name"): runner.app_name,
            get_kagent_metadata_key("user_id"): run_args["user_id"],
            get_kagent_metadata_key("session_id"): run_args["session_id"],
        }

    async def _publish_status_update(
        self,
        event_queue: EventQueue,
        context: RequestContext,
        state: TaskState,
        *,
        message: Message | None = None,
        metadata: dict[str, str] | None = None,
        final: bool = False,
    ):
        """Publish a task status update event."""
        await event_queue.enqueue_event(
            TaskStatusUpdateEvent(
                task_id=context.task_id,
                status=TaskStatus(
                    state=state,
                    message=message,
                    timestamp=datetime.now(timezone.utc).isoformat(),
                ),
                context_id=context.context_id,
                final=final,
                metadata=metadata,
            )
        )

    async def _publish_failure(self, event_queue: EventQueue, context: RequestContext, error: Exception):
        """Publish a task failure event."""
        try:
            await self._publish_status_update(
                event_queue,
                context,
                TaskState.failed,
                message=Message(
                    message_id=str(uuid.uuid4()),
                    role=Role.agent,
                    parts=[Part(TextPart(text=str(error)))],
                ),
                final=True,
            )
        except Exception as e:
            logger.error("Failed to publish failure event: %s", e, exc_info=True)

    async def _publish_final_result(
        self, event_queue: EventQueue, context: RequestContext, aggregator: TaskResultAggregator
    ):
        """Publish the final task result."""
        if (
            aggregator.task_state == TaskState.working
            and aggregator.task_status_message is not None
            and aggregator.task_status_message.parts
        ):
            # Task completed successfully, publish artifact
            await event_queue.enqueue_event(
                TaskArtifactUpdateEvent(
                    task_id=context.task_id,
                    last_chunk=True,
                    context_id=context.context_id,
                    artifact=Artifact(
                        artifact_id=str(uuid.uuid4()),
                        parts=aggregator.task_status_message.parts,
                    ),
                )
            )
            await self._publish_status_update(event_queue, context, TaskState.completed, final=True)
        else:
            # Task failed or has no result
            await self._publish_status_update(
                event_queue,
                context,
                aggregator.task_state,
                message=aggregator.task_status_message,
                final=True,
            )
