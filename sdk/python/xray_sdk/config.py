"""
Configuration management for X-Ray SDK.
"""

from dataclasses import dataclass
from typing import Optional, Literal, Type, Set, Dict, Union
from enum import Enum
import os

from xray_sdk.types import (
    XRayStepType, XRayPipelineID, XRayReasonCode,
    is_xray_step_type, is_xray_pipeline_id, is_xray_reason_code
)


@dataclass
class SamplingConfig:
    """
    Configurable sampling rates for different decision outcomes.

    Allows you to control what percentage of decisions get indexed for each outcome type.
    Rates should be between 0.0 (never sample) and 1.0 (always sample).

    Example:
        SamplingConfig({
            "rejected": 1.0,      # Always index rejections
            "accepted": 0.01,     # Sample 1% of acceptances
            "transformed": 0.1,   # Sample 10% of transformations
            "escalated": 1.0,     # Always index escalations
        })
    """
    outcome_rates: Dict[str, float]

    def should_sample(self, outcome: str, item_id: str) -> bool:
        """
        Determine if a decision should be sampled based on outcome and item ID.

        Uses deterministic hashing for consistent sampling of the same item.
        """
        # Check for specific outcome rate, then wildcard, then default 1%
        rate = self.outcome_rates.get(outcome)
        if rate is None:
            rate = self.outcome_rates.get("*", 0.01)

        if rate >= 1.0:
            return True  # Always sample
        if rate <= 0.0:
            return False  # Never sample

        # Deterministic sampling using item_id hash
        import hashlib
        hash_val = int(hashlib.md5(item_id.encode()).hexdigest(), 16)
        return (hash_val % 100) < int(rate * 100)


@dataclass
class XRayConfig:
    """SDK configuration settings."""
    
    # API connection
    api_key: Optional[str] = None
    endpoint: str = "http://localhost:8080/api/v1"
    timeout_ms: int = 5000
    
    # Buffering
    async_send: bool = True
    batch_size: int = 100
    flush_interval_seconds: float = 1.0
    
    # Fallback
    fallback: Literal["none", "local_file", "memory"] = "memory"
    fallback_path: Optional[str] = None
    
    # Limits
    max_decisions_per_event: int = 10000
    max_item_snapshot_bytes: int = 1024
    max_sample_items: int = 5
    
    # Behavior
    enabled: bool = True
    debug: bool = False


# Global config instance
_config: Optional[XRayConfig] = None

# Registry: pipeline_id -> set of valid step type values
_pipeline_registry: Dict[str, Set[str]] = {}

# Registry: pipeline_id -> set of valid reason code values
_reason_code_registry: Dict[str, Set[str]] = {}

# Global reason codes (not tied to a specific pipeline)
_global_reason_codes: Set[str] = set()


def configure(
    api_key: Optional[str] = None,
    endpoint: Optional[str] = None,
    async_send: bool = True,
    fallback: Literal["none", "local_file", "memory"] = "memory",
    fallback_path: Optional[str] = None,
    timeout_ms: int = 5000,
    batch_size: int = 100,
    flush_interval_seconds: float = 1.0,
    max_decisions_per_event: int = 10000,
    enabled: bool = True,
    debug: bool = False,
) -> XRayConfig:
    """
    Configure the X-Ray SDK.
    
    Args:
        api_key: API key for authentication. Can also be set via XRAY_API_KEY env var.
        endpoint: API endpoint URL. Default: http://localhost:8080/api/v1
        async_send: If True, send data asynchronously in background thread.
        fallback: What to do if API is unreachable: "none", "local_file", or "memory".
        fallback_path: Path for local file fallback.
        timeout_ms: Request timeout in milliseconds.
        batch_size: Number of items to batch before sending.
        flush_interval_seconds: Max time to wait before flushing buffer.
        max_decisions_per_event: Maximum decisions to record per event.
        enabled: If False, SDK becomes a no-op.
        debug: If True, print debug logs.
    
    Returns:
        The configuration object.
    """
    global _config
    
    # Environment variable fallbacks
    resolved_api_key = api_key or os.environ.get("XRAY_API_KEY")
    resolved_endpoint = endpoint or os.environ.get("XRAY_ENDPOINT", "http://localhost:8080/api/v1")
    
    _config = XRayConfig(
        api_key=resolved_api_key,
        endpoint=resolved_endpoint,
        async_send=async_send,
        fallback=fallback,
        fallback_path=fallback_path,
        timeout_ms=timeout_ms,
        batch_size=batch_size,
        flush_interval_seconds=flush_interval_seconds,
        max_decisions_per_event=max_decisions_per_event,
        enabled=enabled,
        debug=debug,
    )
    
    return _config


def get_config() -> XRayConfig:
    """Get the current configuration, creating default if not configured."""
    global _config
    if _config is None:
        _config = XRayConfig()
    return _config


def reset_config() -> None:
    """Reset configuration to None. Mainly for testing."""
    global _config, _pipeline_registry, _reason_code_registry, _global_reason_codes
    _config = None
    _pipeline_registry = {}
    _reason_code_registry = {}
    _global_reason_codes = set()


def register_pipeline(
    pipeline_id: Type[XRayPipelineID],
    step_types: Type[XRayStepType],
    reason_codes: Optional[Type[XRayReasonCode]] = None,
) -> None:
    """
    Register a pipeline with its allowed step types and reason codes.
    
    Args:
        pipeline_id: The pipeline identifier. Should be an XRayPipelineID enum value.
        step_types: An XRayStepType subclass containing valid step types.
        reason_codes: Optional XRayReasonCode subclass for this pipeline's reason codes.
    
    Raises:
        TypeError: If step_types is not an XRayStepType subclass.
        TypeError: If any step type member does not have a string value.
    
    Example:
        from xray_sdk import XRayStepType, XRayPipelineID, XRayReasonCode
        
        class VoiceAgentSteps(XRayStepType):
            STT = "stt"
            LLM = "llm"
            TTS = "tts"
            ACTION_CONTROL = "action_control"
        
        class VoiceAgentPipelines(XRayPipelineID):
            VOICE_AGENT = "voice-agent"
        
        class ActionReasons(XRayReasonCode):
            COOLDOWN_ACTIVE = "COOLDOWN_ACTIVE"
            LIMIT_REACHED = "LIMIT_REACHED"
        
        xray.register_pipeline(
            VoiceAgentPipelines.VOICE_AGENT,
            VoiceAgentSteps,
            ActionReasons
        )
    """
    global _pipeline_registry, _reason_code_registry
    
    # Extract pipeline ID string
    if isinstance(pipeline_id, XRayPipelineID):
        pid = pipeline_id.value
    elif isinstance(pipeline_id, Enum):
        pid = pipeline_id.value
    else:
        pid = str(pipeline_id)
    
    # Validate step_types
    if not is_xray_step_type(step_types):
        raise TypeError(
            f"step_types must be an XRayStepType subclass, got {type(step_types).__name__}. "
            "Define your step types as: class MySteps(XRayStepType): ..."
        )
    
    # Validate that all step type members have string values
    for member in step_types:
        if not isinstance(member.value, str):
            raise TypeError(
                f"Step type '{member.name}' must have a string value, "
                f"got {type(member.value).__name__}: {member.value!r}. "
                f"Define as: {member.name} = \"some_value\""
            )
    
    # Extract step type values
    step_values = {member.value for member in step_types}
    _pipeline_registry[pid] = step_values
    
    # Register reason codes if provided
    if reason_codes is not None:
        if not is_xray_reason_code(reason_codes):
            raise TypeError(
                f"reason_codes must be an XRayReasonCode subclass, got {type(reason_codes).__name__}. "
                "Define your reason codes as: class MyReasons(XRayReasonCode): ..."
            )
        
        # Validate that all reason code members have string values
        for member in reason_codes:
            if not isinstance(member.value, str):
                raise TypeError(
                    f"Reason code '{member.name}' must have a string value, "
                    f"got {type(member.value).__name__}: {member.value!r}. "
                    f"Define as: {member.name} = \"SOME_VALUE\""
                )
        
        reason_values = {member.value for member in reason_codes}
        _reason_code_registry[pid] = reason_values
    
    if get_config().debug:
        print(f"[xray] Registered pipeline '{pid}'")
        print(f"       Step types: {step_values}")
        if reason_codes:
            print(f"       Reason codes: {_reason_code_registry.get(pid, set())}")



def register_reason_codes(reason_codes: Type[XRayReasonCode]) -> None:
    """
    Register global reason codes (not tied to a specific pipeline).
    
    Use this when you have reason codes shared across multiple pipelines.
    
    Args:
        reason_codes: An XRayReasonCode subclass.
    
    Example:
        class CommonReasons(XRayReasonCode):
            RATE_LIMITED = "RATE_LIMITED"
            PERMISSION_DENIED = "PERMISSION_DENIED"
        
        xray.register_reason_codes(CommonReasons)
    """
    global _global_reason_codes
    
    if not is_xray_reason_code(reason_codes):
        raise TypeError(
            f"reason_codes must be an XRayReasonCode subclass, got {type(reason_codes).__name__}. "
            "Define your reason codes as: class MyReasons(XRayReasonCode): ..."
        )
    
    reason_values = {member.value for member in reason_codes}
    _global_reason_codes.update(reason_values)
    
    if get_config().debug:
        print(f"[xray] Registered global reason codes: {reason_values}")


def get_registered_pipelines() -> Dict[str, Set[str]]:
    """Get all registered pipelines and their step types."""
    return _pipeline_registry.copy()


def get_registered_step_types(pipeline_id: Union[XRayPipelineID, str, Enum]) -> Optional[Set[str]]:
    """Get the registered step types for a pipeline."""
    pid = pipeline_id.value if isinstance(pipeline_id, Enum) else str(pipeline_id)
    return _pipeline_registry.get(pid)


def get_registered_reason_codes(pipeline_id: Optional[Union[XRayPipelineID, str, Enum]] = None) -> Set[str]:
    """
    Get registered reason codes.
    
    Args:
        pipeline_id: If provided, returns pipeline-specific codes + global codes.
                    If None, returns only global codes.
    """
    codes = _global_reason_codes.copy()
    if pipeline_id is not None:
        pid = pipeline_id.value if isinstance(pipeline_id, Enum) else str(pipeline_id)
        codes.update(_reason_code_registry.get(pid, set()))
    return codes


def is_pipeline_registered(pipeline_id: Union[XRayPipelineID, str, Enum]) -> bool:
    """Check if a pipeline is registered."""
    pid = pipeline_id.value if isinstance(pipeline_id, Enum) else str(pipeline_id)
    return pid in _pipeline_registry


def validate_step_type(
    pipeline_id: Union[XRayPipelineID, str, Enum],
    step_type: Union[XRayStepType, str, Enum]
) -> None:
    """
    Validate that a step type is registered for a pipeline.
    
    Raises:
        ValueError: If the pipeline is not registered or step type is invalid.
    """
    pid = pipeline_id.value if isinstance(pipeline_id, Enum) else str(pipeline_id)
    st = step_type.value if isinstance(step_type, Enum) else str(step_type)
    
    if pid not in _pipeline_registry:
        raise ValueError(
            f"Pipeline '{pid}' is not registered. "
            f"Call xray.register_pipeline() first."
        )
    
    valid_types = _pipeline_registry[pid]
    if st not in valid_types:
        raise ValueError(
            f"Step type '{st}' is not registered for pipeline '{pid}'. "
            f"Valid step types: {sorted(valid_types)}"
        )


def validate_reason_code(
    reason_code: Union[XRayReasonCode, str, Enum],
    pipeline_id: Optional[Union[XRayPipelineID, str, Enum]] = None,
    strict: bool = False
) -> None:
    """
    Validate that a reason code is registered.
    
    Args:
        reason_code: The reason code to validate.
        pipeline_id: Optional pipeline ID for pipeline-specific validation.
        strict: If True, raises error for unregistered codes. If False, just warns.
    
    Raises:
        ValueError: If strict=True and reason code is not registered.
    """
    rc = reason_code.value if isinstance(reason_code, Enum) else str(reason_code)
    
    valid_codes = get_registered_reason_codes(pipeline_id)
    
    if rc not in valid_codes:
        msg = f"Reason code '{rc}' is not registered."
        if valid_codes:
            msg += f" Valid codes: {sorted(valid_codes)}"
        
        if strict:
            raise ValueError(msg)
        elif get_config().debug:
            print(f"[xray] WARNING: {msg}")
