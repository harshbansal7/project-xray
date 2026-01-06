"""
X-Ray SDK: Reasoning-based observability for decision pipelines.

Setup (one-time):
    import xray_sdk as xray
    from xray_sdk import XRayStepType, XRayPipelineID, XRayReasonCode
    
    # Define your types
    class MyPipelines(XRayPipelineID):
        VOICE_AGENT = "voice-agent"
        COMPETITOR = "competitor-selection"
    
    class MySteps(XRayStepType):
        STT = "stt"
        LLM = "llm"
        TTS = "tts"
        FILTER = "filter"
    
    class MyReasons(XRayReasonCode):
        COOLDOWN_ACTIVE = "COOLDOWN_ACTIVE"
        LIMIT_REACHED = "LIMIT_REACHED"
    
    # Configure and register
    xray.configure(endpoint="https://api.xray.dev")
    xray.register_pipeline(MyPipelines.VOICE_AGENT, MySteps, MyReasons)

Usage:
    with xray.trace(MyPipelines.VOICE_AGENT, metadata={"player": "123"}) as t:
        with t.event("action_control", step_type=MySteps.FILTER, capture="full") as e:
            e.set_input(actions)
            for action in actions:
                allowed, reason = check(action)
                e.record_decision(action, "accepted" if allowed else "rejected",
                                  reason_code=reason)
            e.set_output(allowed_actions)
"""

from xray_sdk.types import (
    XRayStepType,
    XRayPipelineID,
    XRayReasonCode,
)
from xray_sdk.config import (
    configure,
    get_config,
    register_pipeline,
    register_reason_codes,
    get_registered_pipelines,
    get_registered_step_types,
    get_registered_reason_codes,
    is_pipeline_registered,
    SamplingConfig,
)
from xray_sdk.trace import Trace, trace
from xray_sdk.event import Event
from xray_sdk.decision import Decision
from xray_sdk.query import get_trace, query
from xray_sdk.models import CaptureMode

__version__ = "0.1.0"
__all__ = [
    # Base types (users extend these)
    "XRayStepType",
    "XRayPipelineID",
    "XRayReasonCode",
    # Configuration
    "configure",
    "get_config",
    "register_pipeline",
    "register_reason_codes",
    "get_registered_pipelines",
    "get_registered_step_types",
    "get_registered_reason_codes",
    "is_pipeline_registered",
    # Sampling
    "SamplingConfig",
    # Core classes
    "Trace",
    "trace",
    "Event",
    "Decision",
    # Enums (SDK-provided, not user-defined)
    "CaptureMode",
    # Querying
    "get_trace",
    "query",
]
