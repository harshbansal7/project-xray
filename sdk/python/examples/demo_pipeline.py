"""
X-Ray SDK Demo: Image Classification Pipeline

Demonstrates how to trace a multi-step ML pipeline with X-Ray:
- Dataset loading
- Transformations
- Filtering with decision logging
- Model training and inference
"""

import xray_sdk as xray
from xray_sdk import XRayStepType, XRayPipelineID, XRayReasonCode

import numpy as np
from sklearn.datasets import load_digits
from sklearn.model_selection import train_test_split
from sklearn.ensemble import RandomForestClassifier
from sklearn.metrics import accuracy_score
from typing import Tuple, List, Dict

# X-Ray type definitions
class DigitPipelines(XRayPipelineID):
    DIGIT_CLASSIFICATION = "digit-classification"


class DigitSteps(XRayStepType):
    API = "api"
    TRANSFORM = "transform"
    FILTER = "filter"
    LLM = "llm"
    SELECT = "select"


class DigitReasons(XRayReasonCode):
    CONTRAST_OK = "CONTRAST_OK"
    LOW_CONTRAST = "LOW_CONTRAST"
    DIGIT_ALLOWED = "DIGIT_ALLOWED"
    DIGIT_EXCLUDED = "DIGIT_EXCLUDED"
    CORRECT = "CORRECT"
    INCORRECT = "INCORRECT"


# X-Ray configuration
xray.configure(
    endpoint="http://localhost:8080/api/v1",
    async_send=True,
    fallback="local_file",
    fallback_path="./xray_demo_fallback",
)

xray.register_pipeline(
    DigitPipelines.DIGIT_CLASSIFICATION,
    DigitSteps,
    DigitReasons,
)

sampling_config = xray.SamplingConfig({
    DigitReasons.CONTRAST_OK: 0.5,
    DigitReasons.LOW_CONTRAST: 1.0,
    DigitReasons.DIGIT_ALLOWED: 0.1,
    DigitReasons.DIGIT_EXCLUDED: 1.0,
    DigitReasons.CORRECT: 0.05,
    DigitReasons.INCORRECT: 0.8,
})


# Pipeline helpers
def transform_images(images: np.ndarray) -> np.ndarray:
    images = images / 16.0
    noise = np.random.normal(0, 0.05, images.shape)
    return np.clip(images + noise, 0, 1)


def filter_low_contrast(
    images: np.ndarray,
    labels: np.ndarray,
    min_contrast: float,
) -> Tuple[np.ndarray, np.ndarray, List[Dict]]:
    out_images, out_labels, decisions = [], [], []

    for i, (img, label) in enumerate(zip(images, labels)):
        contrast = float(np.std(img))
        accepted = contrast >= min_contrast

        decisions.append({
            "item_id": f"img_{i}",
            "outcome": "accepted" if accepted else "rejected",
            "reason_code": DigitReasons.CONTRAST_OK if accepted else DigitReasons.LOW_CONTRAST,
            "reason_detail": f"contrast={contrast:.3f}",
            "scores": {"contrast": contrast},
        })

        if accepted:
            out_images.append(img)
            out_labels.append(label)

    return np.array(out_images), np.array(out_labels), decisions


def filter_digits(
    images: np.ndarray,
    labels: np.ndarray,
    exclude: List[int],
) -> Tuple[np.ndarray, np.ndarray, List[Dict]]:
    out_images, out_labels, decisions = [], [], []

    for i, (img, label) in enumerate(zip(images, labels)):
        accepted = label not in exclude

        decisions.append({
            "item_id": f"img_{i}",
            "outcome": "accepted" if accepted else "rejected",
            "reason_code": DigitReasons.DIGIT_ALLOWED if accepted else DigitReasons.DIGIT_EXCLUDED,
            "reason_detail": f"digit={label}",
            "scores": {"digit": int(label)},
        })

        if accepted:
            out_images.append(img)
            out_labels.append(label)

    return np.array(out_images), np.array(out_labels), decisions


def classify(
    X_train: np.ndarray,
    y_train: np.ndarray,
    X_test: np.ndarray,
    y_test: np.ndarray,
) -> Tuple[np.ndarray, float, List[Dict]]:
    clf = RandomForestClassifier(n_estimators=50, random_state=42)

    clf.fit(X_train.reshape(len(X_train), -1), y_train)
    probs = clf.predict_proba(X_test.reshape(len(X_test), -1))
    preds = np.argmax(probs, axis=1)

    decisions = []
    for i, (pred, true, p) in enumerate(zip(preds, y_test, probs)):
        correct = pred == true
        decisions.append({
            "item_id": f"test_{i}",
            "outcome": "accepted" if correct else "rejected",
            "reason_code": DigitReasons.CORRECT if correct else DigitReasons.INCORRECT,
            "reason_detail": f"pred={pred}, true={true}",
            "scores": {
                "confidence": float(np.max(p)),
                "predicted": int(pred),
                "actual": int(true),
            },
        })

    return preds, accuracy_score(y_test, preds), decisions


def run_pipeline():
    with xray.trace(
        DigitPipelines.DIGIT_CLASSIFICATION,
        metadata={"model": "RandomForest"},
        sampling_config=sampling_config,
    ) as trace:

        with trace.event("load_dataset", step_type=DigitSteps.API) as e:
            data = load_digits()
            e.set_output(data.images, count=len(data.images))

        with trace.event("transform", step_type=DigitSteps.TRANSFORM) as e:
            images = transform_images(data.images)
            e.set_output(images)

        with trace.event("filter_contrast", step_type=DigitSteps.FILTER, capture="full") as e:
            images, labels, decisions = filter_low_contrast(
                images, data.target, min_contrast=0.36
            )
            for d in decisions:
                e.record_decision(**d)
            e.set_output(images)

        with trace.event("filter_digits", step_type=DigitSteps.FILTER, capture="full") as e:
            images, labels, decisions = filter_digits(images, labels, exclude=[0, 1])
            for d in decisions:
                e.record_decision(**d)
            e.set_output(images)

        with trace.event("split", step_type=DigitSteps.TRANSFORM):
            X_train, X_test, y_train, y_test = train_test_split(
                images, labels, test_size=0.2, random_state=42
            )

        with trace.event("classify", step_type=DigitSteps.LLM, capture="sample") as e:
            preds, acc, decisions = classify(X_train, y_train, X_test, y_test)
            for d in decisions:
                e.record_decision(**d)
            e.annotate("accuracy", acc)

        with trace.event("select_confident", step_type=DigitSteps.SELECT) as e:
            confident = sum(d["scores"]["confidence"] > 0.8 for d in decisions)
            e.set_output(confident)

        return trace.trace_id


if __name__ == "__main__":
    trace_id = run_pipeline()
    print(f"Trace ID: {trace_id}")
