"""
Trace context manager for capturing complete pipeline executions.
"""

from contextlib import contextmanager
from datetime import datetime
from typing import Any, Dict, Generator, List, Optional, Union
from enum import Enum

from xray_sdk.models import TraceData
from xray_sdk.event import Event
from xray_sdk.config import get_config, SamplingConfig
from xray_sdk.client import get_client
from xray_sdk.types import XRayPipelineID, XRayStepType


class Trace:
    """
    Represents a complete pipeline execution.
    
    Use as a context manager to automatically capture timing
    and send data on exit.
    
    Example:
        >>> with xray.trace(MyPipelines.VOICE_AGENT, input=audio) as t:
        ...     with t.event("stt", step_type=MySteps.STT) as e:
        ...         text = transcribe(audio)
        ...         e.set_output(text)
    """
    
    def __init__(
        self,
        pipeline_id: Union[XRayPipelineID, str, Enum],
        input_data: Any = None,
        metadata: Optional[Dict[str, Any]] = None,
        tags: Optional[List[str]] = None,
        sampling_config: Optional[SamplingConfig] = None,
    ):
        """
        Create a new Trace.
        
        Args:
            pipeline_id: Pipeline identifier (XRayPipelineID enum value recommended).
            input_data: The original input to the pipeline.
            metadata: Arbitrary metadata to attach (user_id, session, etc.)
            tags: Tags for filtering (e.g., ["v2", "experiment-a"]).
            sampling_config: Configurable sampling rates per outcome type.
        """
        # Convert enum to string
        pipeline_id_str = pipeline_id.value if isinstance(pipeline_id, Enum) else str(pipeline_id)
        
        self._data = TraceData(
            pipeline_id=pipeline_id_str,
            input_data=input_data,
            metadata=metadata or {},
            tags=tags or [],
        )
        self._pipeline_id_original = pipeline_id  # Keep original for validation
        self._sampling_config = sampling_config
        self._client = get_client()
        self._events: List[Event] = []
        self._config = get_config()
    
    def __enter__(self) -> "Trace":
        self._data.started_at = datetime.utcnow()
        self._data.status = "running"
        
        if self._client and self._config.enabled:
            self._client.queue_trace_start(self._data)
        
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb) -> bool:
        self._data.ended_at = datetime.utcnow()
        self._data.status = "failed" if exc_type else "completed"
        
        if self._client and self._config.enabled:
            self._client.queue_trace_end(self._data)
        
        return False
    
    @property
    def trace_id(self) -> str:
        """Get the unique trace ID."""
        return self._data.trace_id
    
    @property
    def pipeline_id(self) -> str:
        """Get the pipeline ID."""
        return self._data.pipeline_id
    
    @contextmanager
    def event(
        self,
        step_name: str,
        step_type: Union[XRayStepType, str, Enum],
        capture: str = "metrics",
        sampling_config: Optional[SamplingConfig] = None,
        **annotations,
    ) -> Generator[Event, None, None]:
        """
        Create a new event within this trace.
        
        Args:
            step_name: Human-readable name for this step.
            step_type: XRayStepType enum value (recommended) or string.
            capture: How much detail to capture: "metrics", "sample", or "full".
            sampling_config: Override trace-level sampling config for this event.
            **annotations: Additional annotations to attach.
        
        Yields:
            Event context manager.
        """
        # Use event-level sampling config, or fall back to trace-level
        event_sampling_config = sampling_config or self._sampling_config

        evt = Event(
            trace_id=self._data.trace_id,
            step_name=step_name,
            step_type=step_type,
            pipeline_id=self._pipeline_id_original,
            capture=capture,
            annotations=annotations if annotations else None,
            client=self._client,
            sampling_config=event_sampling_config,
        )
        self._events.append(evt)
        
        with evt:
            yield evt
    
    def add_metadata(self, key: str, value: Any) -> None:
        """Add metadata to the trace."""
        self._data.metadata[key] = value
    
    def add_tag(self, tag: str) -> None:
        """Add a tag to the trace."""
        if tag not in self._data.tags:
            self._data.tags.append(tag)
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization."""
        return self._data.model_dump(mode="json")
    
    def __repr__(self) -> str:
        return f"Trace(pipeline_id={self._data.pipeline_id!r}, trace_id={self._data.trace_id!r})"


@contextmanager
def trace(
    pipeline_id: Union[XRayPipelineID, str, Enum],
    input_data: Any = None,
    metadata: Optional[Dict[str, Any]] = None,
    tags: Optional[List[str]] = None,
    sampling_config: Optional[SamplingConfig] = None,
) -> Generator[Trace, None, None]:
    """
    Create a new trace for a pipeline execution.
    
    This is the main entry point for X-Ray instrumentation.
    
    Args:
        pipeline_id: Pipeline identifier (XRayPipelineID enum value recommended).
        input_data: The original input to the pipeline.
        metadata: Arbitrary metadata to attach.
        tags: Tags for filtering.
        sampling_config: Configurable sampling rates per outcome type.
    
    Yields:
        Trace context manager.
    
    Example:
        >>> from xray_sdk import SamplingConfig
        >>> 
        >>> # Custom sampling: always index rejections, sample 10% of acceptances
        >>> sampling = SamplingConfig({"rejected": 1.0, "accepted": 0.1})
        >>>
        >>> with xray.trace(MyPipelines.VOICE_AGENT, sampling_config=sampling) as t:
        ...     with t.event("action_control", step_type=MySteps.FILTER) as e:
        ...         e.set_input(available_actions)
        ...         for action in available_actions:
        ...             allowed, reason = check_action(action)
        ...             e.record_decision(action, "accepted" if allowed else "rejected",
        ...                               reason_code=reason.code)
        ...         e.set_output(allowed_actions)
    """
    t = Trace(
        pipeline_id=pipeline_id,
        input_data=input_data,
        metadata=metadata,
        tags=tags,
        sampling_config=sampling_config,
    )
    with t:
        yield t
