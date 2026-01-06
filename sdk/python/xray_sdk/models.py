"""
Data models for X-Ray SDK using Pydantic.
"""

from datetime import datetime
from typing import Any, Dict, List, Optional, Union
from enum import Enum
from pydantic import BaseModel, Field
import uuid


# Note: StepType is no longer a hardcoded enum.
# Developers define their own step type enums and register them with xray.register_pipeline()


class CaptureMode(str, Enum):
    """How much detail to capture for an event."""
    METRICS = "metrics"  # Only counts and timing (default)
    SAMPLE = "sample"    # 1% of decisions
    FULL = "full"        # All decisions


class DecisionData(BaseModel):
    """Data model for an individual item decision."""
    decision_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    event_id: str
    trace_id: str
    item_id: str
    outcome: str
    reason_code: Optional[str] = None
    reason_detail: Optional[str] = None
    scores: Dict[str, float] = Field(default_factory=dict)
    item_snapshot: Optional[Dict[str, Any]] = None
    timestamp: datetime = Field(default_factory=datetime.utcnow)


class EventMetrics(BaseModel):
    """Computed metrics for an event."""
    input_count: Optional[int] = None
    output_count: Optional[int] = None
    reduction_ratio: Optional[float] = None
    duration_ms: Optional[float] = None


class EventData(BaseModel):
    """Data model for a pipeline step/event."""
    event_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    trace_id: str
    parent_event_id: Optional[str] = None
    step_name: str
    step_type: str  # Now a string - validated against registry in Event class
    capture_mode: CaptureMode = CaptureMode.METRICS
    
    input_count: Optional[int] = None
    input_sample: Optional[List[Any]] = None
    output_count: Optional[int] = None
    output_sample: Optional[List[Any]] = None
    
    metrics: EventMetrics = Field(default_factory=EventMetrics)
    annotations: Dict[str, Any] = Field(default_factory=dict)
    
    started_at: datetime = Field(default_factory=datetime.utcnow)
    ended_at: Optional[datetime] = None


class TraceData(BaseModel):
    """Data model for a complete pipeline execution."""
    trace_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    pipeline_id: str
    started_at: datetime = Field(default_factory=datetime.utcnow)
    ended_at: Optional[datetime] = None
    metadata: Dict[str, Any] = Field(default_factory=dict)
    input_data: Optional[Any] = None
    tags: List[str] = Field(default_factory=list)
    status: str = "running"  # running, completed, failed


class TraceWithEvents(BaseModel):
    """A trace with all its events and decisions."""
    trace: TraceData
    events: List[EventData] = Field(default_factory=list)
    decisions: Dict[str, List[DecisionData]] = Field(default_factory=dict)  # event_id -> decisions

