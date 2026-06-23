from __future__ import annotations

from dataclasses import asdict, dataclass

import numpy as np
from sklearn.metrics import (
    accuracy_score,
    classification_report,
    f1_score,
    precision_score,
    recall_score,
    roc_auc_score,
)


@dataclass
class ModelResult:
    name: str
    accuracy: float
    precision: float
    recall: float
    f1: float
    roc_auc: float

    def to_dict(self) -> dict:
        return asdict(self)


def evaluate(name: str, y_true, y_pred, y_proba=None) -> ModelResult:
    roc = float("nan")
    if y_proba is not None and len(np.unique(y_true)) == 2:
        roc = roc_auc_score(y_true, y_proba)
    return ModelResult(
        name=name,
        accuracy=accuracy_score(y_true, y_pred),
        precision=precision_score(y_true, y_pred, zero_division=0),
        recall=recall_score(y_true, y_pred, zero_division=0),
        f1=f1_score(y_true, y_pred, zero_division=0),
        roc_auc=roc,
    )


def format_report(y_true, y_pred) -> str:
    return classification_report(y_true, y_pred, target_names=["<=50K", ">50K"])
