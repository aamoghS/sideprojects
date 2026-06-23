"""Random Forest variants and LightGBM wrappers."""

from __future__ import annotations

from sklearn.base import BaseEstimator, ClassifierMixin
from sklearn.ensemble import (
    ExtraTreesClassifier,
    RandomForestClassifier,
)
from lightgbm import LGBMClassifier

from src.config import BASELINE_LGBM, BASELINE_RF, BASELINE_TREE, RANDOM_STATE


def baseline_random_forest(**overrides) -> RandomForestClassifier:
    params = {**BASELINE_RF, "random_state": RANDOM_STATE, "n_jobs": -1}
    params.update(overrides)
    return RandomForestClassifier(**params)


def baseline_lightgbm(**overrides) -> LGBMClassifier:
    params = {
        **BASELINE_LGBM,
        "random_state": RANDOM_STATE,
        "n_jobs": -1,
        "verbose": -1,
    }
    params.update(overrides)
    return LGBMClassifier(**params)


def extra_trees(**overrides) -> ExtraTreesClassifier:
    params = {**BASELINE_TREE, "random_state": RANDOM_STATE, "n_jobs": -1}
    params.update(overrides)
    return ExtraTreesClassifier(**params)


def balanced_random_forest(**overrides) -> RandomForestClassifier:
    """RF with class_weight=balanced for skewed labels."""
    params = {**BASELINE_TREE, "random_state": RANDOM_STATE, "n_jobs": -1, "class_weight": "balanced"}
    params.update(overrides)
    return RandomForestClassifier(**params)


def bootstrap_false_forest(**overrides) -> RandomForestClassifier:
    """RF without bagging — each tree sees the full training set."""
    params = {
        **BASELINE_RF,
        "random_state": RANDOM_STATE,
        "n_jobs": -1,
        "bootstrap": False,
    }
    params.update(overrides)
    return RandomForestClassifier(**params)


RF_VARIANTS: dict[str, type[BaseEstimator] | callable] = {
    "random_forest": baseline_random_forest,
    "extra_trees": extra_trees,
    "balanced_rf": balanced_random_forest,
    "bootstrap_false_rf": bootstrap_false_forest,
}


def make_rf_variant(name: str, **params) -> ClassifierMixin:
    factory = RF_VARIANTS[name]
    return factory(**params)
