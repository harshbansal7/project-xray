"""
Decision recording for item-level visibility.
"""

from typing import Any, Dict, Optional
from xray_sdk.models import DecisionData


class Decision:
    """
    Represents an individual item decision within an event.
    
    A Decision records what happened to a single item (accepted, rejected, 
    or transformed) and why. This enables debugging at the item level.
    
    Example:
        >>> decision = Decision(
        ...     event_id="evt_123",
        ...     trace_id="trace_456", 
        ...     item_id="ASIN-B08N5W",
        ...     outcome="rejected",
        ...     reason_code="PRICE_TOO_HIGH",
        ...     reason_detail="Price $89.99 exceeds threshold $60.00",
        ...     scores={"price_ratio": 1.5}
        ... )
    """
    
    def __init__(
        self,
        event_id: str,
        trace_id: str,
        item_id: str,
        outcome: str,
        reason_code: Optional[str] = None,
        reason_detail: Optional[str] = None,
        scores: Optional[Dict[str, float]] = None,
        item_snapshot: Optional[Dict[str, Any]] = None,
    ):
        """
        Create a new Decision.
        
        Args:
            event_id: The event this decision belongs to.
            trace_id: The trace this decision belongs to.
            item_id: Your domain identifier for the item (ASIN, user_id, etc.)
            outcome: One of "accepted", "rejected", or "transformed".
            reason_code: Machine-queryable code (e.g., "PRICE_TOO_HIGH").
            reason_detail: Human-readable explanation.
            scores: Numeric scores if applicable.
            item_snapshot: State of item at decision time (will be truncated).
        """
        self._data = DecisionData(
            event_id=event_id,
            trace_id=trace_id,
            item_id=item_id,
            outcome=outcome,  # Now accepts any string outcome
            reason_code=reason_code,
            reason_detail=reason_detail,
            scores=scores or {},
            item_snapshot=item_snapshot,
        )
    
    @property
    def decision_id(self) -> str:
        return self._data.decision_id
    
    @property
    def item_id(self) -> str:
        return self._data.item_id
    
    @property
    def outcome(self) -> str:
        return self._data.outcome.value
    
    @property
    def reason_code(self) -> Optional[str]:
        return self._data.reason_code
    
    @property
    def reason_detail(self) -> Optional[str]:
        return self._data.reason_detail
    
    @property
    def scores(self) -> Dict[str, float]:
        return self._data.scores
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for serialization."""
        return self._data.model_dump(mode="json")
    
    def __repr__(self) -> str:
        return (
            f"Decision(item_id={self.item_id!r}, outcome={self.outcome!r}, "
            f"reason_code={self.reason_code!r})"
        )
