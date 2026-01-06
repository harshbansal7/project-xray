"""
Event context manager for capturing pipeline steps.
"""

from datetime import datetime
from typing import Any, Dict, List, Optional, Union, TYPE_CHECKING
from enum import Enum
import json
import hashlib

from xray_sdk.models import EventData, EventMetrics, CaptureMode
from xray_sdk.decision import Decision
from xray_sdk.config import get_config, is_pipeline_registered, validate_step_type, validate_reason_code, SamplingConfig
from xray_sdk.types import XRayStepType, XRayPipelineID, XRayReasonCode

if TYPE_CHECKING:
    from xray_sdk.client import XRayClient


class Event:
    """
    Represents a single step in a pipeline.
    
    Use as a context manager within a Trace to automatically capture
    timing and send data on exit.
    
    Example:
        >>> with trace.event("filter_products", step_type=MySteps.FILTER) as e:
        ...     e.set_input(candidates)
        ...     filtered = filter_products(candidates)
        ...     e.set_output(filtered)
    """
    
    def __init__(
        self,
        trace_id: str,
        step_name: str,
        step_type: Union[XRayStepType, str, Enum],
        pipeline_id: Optional[Union[XRayPipelineID, str, Enum]] = None,
        capture: str = "metrics",
        parent_event_id: Optional[str] = None,
        annotations: Optional[Dict[str, Any]] = None,
        client: Optional["XRayClient"] = None,
        sampling_config: Optional[SamplingConfig] = None,
    ):
        """
        Create a new Event.

        Args:
            trace_id: The trace this event belongs to.
            step_name: Human-readable name for this step.
            step_type: Step type (XRayStepType enum value recommended).
            pipeline_id: Pipeline ID for validation.
            capture: How much to capture: "metrics", "sample", or "full".
            parent_event_id: For nested events, the parent event ID.
            annotations: Initial annotations to attach.
            client: X-Ray client for sending data.
            sampling_config: Custom sampling config for this event (overrides trace-level).
        """
        # Convert enum to string value
        step_type_str = step_type.value if isinstance(step_type, Enum) else str(step_type)
        
        # Validate step_type against registry if pipeline is registered
        if pipeline_id is not None:
            pid = pipeline_id.value if isinstance(pipeline_id, Enum) else str(pipeline_id)
            if is_pipeline_registered(pid):
                validate_step_type(pid, step_type_str)
        
        # Validate capture mode
        try:
            capture_enum = CaptureMode(capture)
        except ValueError:
            valid = [c.value for c in CaptureMode]
            raise ValueError(f"capture must be one of {valid}, got '{capture}'")
        
        self._data = EventData(
            trace_id=trace_id,
            step_name=step_name,
            step_type=step_type_str,
            capture_mode=capture_enum,
            parent_event_id=parent_event_id,
            annotations=annotations or {},
        )
        self._client = client
        self._decisions: List[Decision] = []
        self._config = get_config()
        self._pipeline_id = pipeline_id
        self._sampling_config = sampling_config
    
    def __enter__(self) -> "Event":
        self._data.started_at = datetime.utcnow()
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb) -> bool:
        self._data.ended_at = datetime.utcnow()
        self._compute_metrics()
        
        # Send event data
        if self._client and self._config.enabled:
            self._client.queue_event(self._data)
            
            # Send decisions if any
            if self._decisions:
                self._client.queue_decisions(self._data.event_id, self._decisions)
        
        # Don't suppress exceptions
        return False
    
    @property
    def event_id(self) -> str:
        """Get the unique event ID."""
        return self._data.event_id
    
    @property
    def step_name(self) -> str:
        """Get the step name."""
        return self._data.step_name
    
    def set_input(self, data: Any, count: Optional[int] = None) -> None:
        """
        Record input data for this event.
        
        Args:
            data: The input data. If a list/tuple/array, count is auto-computed.
            count: Explicit count if data is not iterable.
        """
        if count is not None:
            self._data.input_count = count
        elif hasattr(data, '__len__'):
            self._data.input_count = len(data)
            if isinstance(data, (list, tuple)):
                self._data.input_sample = self._sample(data)
        else:
            self._data.input_count = 1
    
    def set_output(self, data: Any, count: Optional[int] = None) -> None:
        """
        Record output data for this event.
        
        Args:
            data: The output data. If a list/tuple/array, count is auto-computed.
            count: Explicit count if data is not iterable.
        """
        if count is not None:
            self._data.output_count = count
        elif hasattr(data, '__len__'):
            self._data.output_count = len(data)
            if isinstance(data, (list, tuple)):
                self._data.output_sample = self._sample(data)
        else:
            self._data.output_count = 1
    
    def annotate(self, key: str, value: Any) -> None:
        """
        Add a custom annotation to this event.
        
        Args:
            key: Annotation key.
            value: Annotation value (must be JSON-serializable).
        """
        self._data.annotations[key] = value
    
    def record_decision(
        self,
        item_id: str,
        outcome: str,
        reason_code: Optional[Union[XRayReasonCode, str, Enum]] = None,
        reason_detail: Optional[str] = None,
        scores: Optional[Dict[str, float]] = None,
        item_snapshot: Optional[Dict[str, Any]] = None,
    ) -> None:
        """
        Record a decision for a single item.
        
        This is only effective when capture mode is "sample" or "full".
        In "metrics" mode, decisions are not recorded.
        
        Args:
            item_id: Your domain identifier (ASIN, user_id, action_name, etc.)
            outcome: One of "accepted", "rejected", "transformed".
            reason_code: XRayReasonCode enum value (recommended) or string.
            reason_detail: Human-readable explanation.
            scores: Numeric scores if applicable.
            item_snapshot: State of item at decision time.
        
        Example:
            >>> e.record_decision(
            ...     item_id="GIVE_MONEY",
            ...     outcome="rejected",
            ...     reason_code=ActionReasons.COOLDOWN_ACTIVE,
            ...     reason_detail="Player received money 12 min ago, cooldown is 30 min",
            ...     scores={"minutes_since_last": 12, "cooldown_minutes": 30}
            ... )
        """
        # Determine if we should sample this decision
        should_sample = True

        if self._sampling_config:
            # Use custom sampling config (highest priority)
            should_sample = self._sampling_config.should_sample(outcome, item_id)
        elif self._data.capture_mode == CaptureMode.METRICS:
            # Metrics mode: never sample decisions
            should_sample = False
        elif self._data.capture_mode == CaptureMode.SAMPLE:
            # Legacy sample mode: 1% sampling
            hash_val = int(hashlib.md5(item_id.encode()).hexdigest(), 16)
            should_sample = (hash_val % 100 == 0)
        # FULL mode: should_sample remains True (sample all)

        if not should_sample:
            return
        
        # Check limit
        if len(self._decisions) >= self._config.max_decisions_per_event:
            return
        
        # Convert reason_code enum to string
        reason_code_str: Optional[str] = None
        if reason_code is not None:
            reason_code_str = reason_code.value if isinstance(reason_code, Enum) else str(reason_code)
            # Validate reason code (non-strict: just warns in debug mode)
            validate_reason_code(reason_code_str, self._pipeline_id, strict=False)
        
        # Truncate snapshot if too large
        truncated_snapshot = self._truncate_snapshot(item_snapshot)
        
        decision = Decision(
            event_id=self._data.event_id,
            trace_id=self._data.trace_id,
            item_id=item_id,
            outcome=outcome,
            reason_code=reason_code_str,
            reason_detail=reason_detail,
            scores=scores,
            item_snapshot=truncated_snapshot,
        )
        self._decisions.append(decision)
    
    def _compute_metrics(self) -> None:
        """Compute derived metrics for this event."""
        if self._data.started_at and self._data.ended_at:
            delta = self._data.ended_at - self._data.started_at
            self._data.metrics.duration_ms = delta.total_seconds() * 1000
        
        self._data.metrics.input_count = self._data.input_count
        self._data.metrics.output_count = self._data.output_count
        
        if self._data.input_count and self._data.output_count:
            if self._data.input_count > 0:
                self._data.metrics.reduction_ratio = round(
                    1 - (self._data.output_count / self._data.input_count), 4
                )
    
    def _sample(self, data: List[Any], max_items: Optional[int] = None) -> List[Any]:
        """Sample items for storage."""
        max_items = max_items or self._config.max_sample_items
        if len(data) <= max_items:
            return self._serialize_sample(data)
        
        step = len(data) // max_items
        sampled = [data[i] for i in range(0, len(data), step)][:max_items]
        return self._serialize_sample(sampled)
    
    def _serialize_sample(self, data: List[Any]) -> List[Any]:
        """Ensure sample items are JSON-serializable."""
        result = []
        for item in data:
            if hasattr(item, "model_dump"):
                result.append(item.model_dump())
            elif hasattr(item, "__dict__"):
                result.append(item.__dict__)
            else:
                result.append(item)
        return result
    
    def _truncate_snapshot(self, snapshot: Optional[Dict[str, Any]]) -> Optional[Dict[str, Any]]:
        """Truncate snapshot if too large."""
        if snapshot is None:
            return None
        
        try:
            serialized = json.dumps(snapshot)
            if len(serialized) <= self._config.max_item_snapshot_bytes:
                return snapshot
            
            return {
                "_truncated": True,
                "_original_size": len(serialized),
                "preview": serialized[:self._config.max_item_snapshot_bytes]
            }
        except (TypeError, ValueError):
            return {"_error": "Could not serialize snapshot"}
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization."""
        return self._data.model_dump(mode="json")
    
    def __repr__(self) -> str:
        return f"Event(step_name={self._data.step_name!r}, step_type={self._data.step_type!r})"
