"""
Query API for retrieving traces and events.
"""

from typing import Any, Dict, List, Optional
from datetime import datetime

import httpx

from xray_sdk.config import get_config
from xray_sdk.models import TraceWithEvents, TraceData, EventData


def get_trace(trace_id: str) -> Optional[TraceWithEvents]:
    """
    Retrieve a complete trace with all its events.
    
    Args:
        trace_id: The trace ID to retrieve.
    
    Returns:
        TraceWithEvents object or None if not found.
    
    Example:
        >>> trace = xray.get_trace("abc123")
        >>> print(trace.trace.pipeline_id)
        'competitor-selection'
        >>> for event in trace.events:
        ...     print(f"{event.step_name}: {event.metrics.input_count} → {event.metrics.output_count}")
    """
    config = get_config()
    
    headers = {}
    if config.api_key:
        headers["Authorization"] = f"Bearer {config.api_key}"
    
    try:
        with httpx.Client(base_url=config.endpoint, headers=headers) as client:
            response = client.get(f"/traces/{trace_id}")
            response.raise_for_status()
            
            data = response.json()
            return TraceWithEvents.model_validate(data)
    except httpx.HTTPStatusError as e:
        if e.response.status_code == 404:
            return None
        raise
    except httpx.RequestError as e:
        raise ConnectionError(f"Failed to connect to X-Ray API: {e}") from e


def query(
    pipeline_id: Optional[str] = None,
    step_type: Optional[str] = None,
    min_reduction_ratio: Optional[float] = None,
    time_range: Optional[str] = None,
    start_time: Optional[datetime] = None,
    end_time: Optional[datetime] = None,
    tags: Optional[List[str]] = None,
    limit: int = 100,
    cursor: Optional[str] = None,
) -> Dict[str, Any]:
    """
    Query traces and events with filters.
    
    Args:
        pipeline_id: Filter by pipeline ID.
        step_type: Filter events by step type.
        min_reduction_ratio: Filter events where reduction_ratio >= this value.
        time_range: Shorthand like "last_24h", "last_7d", "today".
        start_time: Start of time range.
        end_time: End of time range.
        tags: Filter by tags (any match).
        limit: Maximum results to return.
        cursor: Pagination cursor from previous query.
    
    Returns:
        Dict with 'results' list and optional 'next_cursor'.
    
    Example:
        >>> # Find all filter steps with >90% reduction
        >>> results = xray.query(
        ...     step_type="filter",
        ...     min_reduction_ratio=0.9,
        ...     time_range="last_7d"
        ... )
        >>> for event in results["results"]:
        ...     print(f"{event['step_name']}: {event['metrics']['reduction_ratio']:.1%}")
        
        >>> # Find traces for a specific pipeline
        >>> results = xray.query(
        ...     pipeline_id="competitor-selection",
        ...     start_time=datetime(2024, 1, 1),
        ...     end_time=datetime(2024, 1, 2)
        ... )
    """
    config = get_config()
    
    headers = {}
    if config.api_key:
        headers["Authorization"] = f"Bearer {config.api_key}"
    
    # Build query params
    params: Dict[str, Any] = {"limit": limit}
    
    if pipeline_id:
        params["pipeline_id"] = pipeline_id
    if step_type:
        params["step_type"] = step_type
    if min_reduction_ratio is not None:
        params["min_reduction_ratio"] = min_reduction_ratio
    if time_range:
        params["time_range"] = time_range
    if start_time:
        params["start_time"] = start_time.isoformat()
    if end_time:
        params["end_time"] = end_time.isoformat()
    if tags:
        params["tags"] = ",".join(tags)
    if cursor:
        params["cursor"] = cursor
    
    try:
        with httpx.Client(base_url=config.endpoint, headers=headers) as client:
            response = client.get("/query", params=params)
            response.raise_for_status()
            return response.json()
    except httpx.RequestError as e:
        raise ConnectionError(f"Failed to connect to X-Ray API: {e}") from e


def get_decisions(
    trace_id: str,
    event_id: str,
    outcome: Optional[str] = None,
    limit: int = 100,
    cursor: Optional[str] = None,
) -> Dict[str, Any]:
    """
    Get decisions for a specific event.
    
    Args:
        trace_id: The trace ID.
        event_id: The event ID.
        outcome: Filter by outcome ("accepted", "rejected", "transformed").
        limit: Maximum results.
        cursor: Pagination cursor.
    
    Returns:
        Dict with 'decisions' list and optional 'next_cursor'.
    
    Example:
        >>> decisions = xray.get_decisions(
        ...     trace_id="abc123",
        ...     event_id="evt456",
        ...     outcome="rejected"
        ... )
        >>> for d in decisions["decisions"]:
        ...     print(f"{d['item_id']}: {d['reason_code']}")
    """
    config = get_config()
    
    headers = {}
    if config.api_key:
        headers["Authorization"] = f"Bearer {config.api_key}"
    
    params: Dict[str, Any] = {"limit": limit}
    if outcome:
        params["outcome"] = outcome
    if cursor:
        params["cursor"] = cursor
    
    try:
        with httpx.Client(base_url=config.endpoint, headers=headers) as client:
            response = client.get(
                f"/traces/{trace_id}/events/{event_id}/decisions",
                params=params
            )
            response.raise_for_status()
            return response.json()
    except httpx.RequestError as e:
        raise ConnectionError(f"Failed to connect to X-Ray API: {e}") from e


def get_item_history(item_id: str, limit: int = 100) -> Dict[str, Any]:
    """
    Get all decisions for an item across all traces.
    
    Args:
        item_id: Your domain identifier (ASIN, user_id, etc.)
        limit: Maximum results.
    
    Returns:
        Dict with 'decisions' list showing item's history.
    
    Example:
        >>> # Where has this product appeared before?
        >>> history = xray.get_item_history("ASIN-B08N5W")
        >>> for d in history["decisions"]:
        ...     print(f"Trace {d['trace_id']}: {d['outcome']}")
    """
    config = get_config()
    
    headers = {}
    if config.api_key:
        headers["Authorization"] = f"Bearer {config.api_key}"
    
    try:
        with httpx.Client(base_url=config.endpoint, headers=headers) as client:
            response = client.get(f"/items/{item_id}/history", params={"limit": limit})
            response.raise_for_status()
            return response.json()
    except httpx.RequestError as e:
        raise ConnectionError(f"Failed to connect to X-Ray API: {e}") from e
