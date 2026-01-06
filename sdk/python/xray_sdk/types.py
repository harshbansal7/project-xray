"""
X-Ray SDK Type System.

These are base types that users MUST extend to define their own:
- Step types (what kind of step: filter, llm, transform, etc.)
- Pipeline IDs (which pipeline: competitor-selection, voice-agent, etc.)
- Reason codes (why rejected: PRICE_TOO_HIGH, COOLDOWN_ACTIVE, etc.)

This enforces type-safety and ensures consistent, queryable values.

Example:
    from xray_sdk import XRayStepType, XRayPipelineID, XRayReasonCode
    
    class MySteps(XRayStepType):
        FILTER = "filter"
        LLM = "llm"
        TRANSFORM = "transform"
    
    class MyPipelines(XRayPipelineID):
        VOICE_AGENT = "voice-agent"
        COMPETITOR = "competitor-selection"
    
    class MyReasonCodes(XRayReasonCode):
        PRICE_TOO_HIGH = "PRICE_TOO_HIGH"
        COOLDOWN_ACTIVE = "COOLDOWN_ACTIVE"
"""

from enum import Enum
from typing import Type, TypeVar


class XRayStepType(str, Enum):
    """
    Base class for defining step types.
    
    Extend this to define the valid step types for your pipelines.
    Step types categorize what a step does (filter, transform, llm call, etc.)
    
    Example:
        class VoiceAgentSteps(XRayStepType):
            STT = "stt"           # Speech-to-text
            LLM = "llm"           # Language model
            TTS = "tts"           # Text-to-speech
            FILTER = "filter"     # Action filtering
    """
    pass


class XRayPipelineID(str, Enum):
    """
    Base class for defining pipeline identifiers.
    
    Extend this to define your organization's pipelines.
    This ensures consistent pipeline naming across your codebase.
    
    Example:
        class MyPipelines(XRayPipelineID):
            VOICE_AGENT = "voice-agent"
            COMPETITOR_SELECTION = "competitor-selection"
            CATEGORIZATION = "categorization"
    """
    pass


class XRayReasonCode(str, Enum):
    """
    Base class for defining rejection/acceptance reason codes.
    
    Extend this to define semantic, queryable reason codes.
    These should be machine-readable codes, not human descriptions.
    
    Example:
        class ActionReasonCodes(XRayReasonCode):
            COOLDOWN_ACTIVE = "COOLDOWN_ACTIVE"
            LIMIT_REACHED = "LIMIT_REACHED"
            PERMISSION_DENIED = "PERMISSION_DENIED"
            
        class FilterReasonCodes(XRayReasonCode):
            PRICE_TOO_HIGH = "PRICE_TOO_HIGH"
            LOW_RATING = "LOW_RATING"
            CATEGORY_MISMATCH = "CATEGORY_MISMATCH"
    """
    pass


# Type vars for generic typing
StepTypeT = TypeVar('StepTypeT', bound=XRayStepType)
PipelineIDT = TypeVar('PipelineIDT', bound=XRayPipelineID)
ReasonCodeT = TypeVar('ReasonCodeT', bound=XRayReasonCode)


def is_xray_step_type(cls: Type) -> bool:
    """Check if a class is a valid XRayStepType subclass."""
    return isinstance(cls, type) and issubclass(cls, XRayStepType) and cls is not XRayStepType


def is_xray_pipeline_id(cls: Type) -> bool:
    """Check if a class is a valid XRayPipelineID subclass."""
    return isinstance(cls, type) and issubclass(cls, XRayPipelineID) and cls is not XRayPipelineID


def is_xray_reason_code(cls: Type) -> bool:
    """Check if a class is a valid XRayReasonCode subclass."""
    return isinstance(cls, type) and issubclass(cls, XRayReasonCode) and cls is not XRayReasonCode

