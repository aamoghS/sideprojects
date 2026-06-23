"""Train deliberately weak baseline models (~60% accuracy)."""

from __future__ import annotations

import json
from pathlib import Path

import joblib
from sklearn.pipeline import Pipeline

from src.config import ARTIFACTS_DIR, RANDOM_STATE
from src.data import load_baseline_dataset, load_dataset
from src.metrics import ModelResult, evaluate, format_report
from src.models import RF_VARIANTS, baseline_lightgbm, baseline_random_forest


def _proba_positive(estimator: Pipeline, X) -> list[float] | None:
    if not hasattr(estimator, "predict_proba"):
        return None
    proba = estimator.predict_proba(X)
    if proba.shape[1] == 2:
        return proba[:, 1]
    return None


def train_baseline_rf() -> tuple[Pipeline, ModelResult]:
    ds = load_baseline_dataset()
    pipe = Pipeline(
        [
            ("prep", ds.preprocessor),
            ("clf", baseline_random_forest()),
        ]
    )
    pipe.fit(ds.X_train, ds.y_train)
    preds = pipe.predict(ds.X_test)
    proba = _proba_positive(pipe, ds.X_test)
    result = evaluate("baseline_rf", ds.y_test, preds, proba)
    return pipe, result


def train_baseline_lgbm() -> tuple[Pipeline, ModelResult]:
    ds = load_baseline_dataset()
    pipe = Pipeline(
        [
            ("prep", ds.preprocessor),
            ("clf", baseline_lightgbm()),
        ]
    )
    pipe.fit(ds.X_train, ds.y_train)
    preds = pipe.predict(ds.X_test)
    proba = _proba_positive(pipe, ds.X_test)
    result = evaluate("baseline_lgbm", ds.y_test, preds, proba)
    return pipe, result


def train_rf_variants() -> list[tuple[str, Pipeline, ModelResult]]:
    ds = load_baseline_dataset()
    results: list[tuple[str, Pipeline, ModelResult]] = []
    for name, factory in RF_VARIANTS.items():
        pipe = Pipeline([("prep", ds.preprocessor), ("clf", factory())])
        pipe.fit(ds.X_train, ds.y_train)
        preds = pipe.predict(ds.X_test)
        proba = _proba_positive(pipe, ds.X_test)
        result = evaluate(name, ds.y_test, preds, proba)
        results.append((name, pipe, result))
    return results


def run_baselines(save: bool = True) -> list[ModelResult]:
    ARTIFACTS_DIR.mkdir(parents=True, exist_ok=True)
    all_results: list[ModelResult] = []

    pipe_rf, rf_result = train_baseline_rf()
    all_results.append(rf_result)
    print(f"\n{rf_result.name}: accuracy={rf_result.accuracy:.3f}")

    pipe_lgbm, lgbm_result = train_baseline_lgbm()
    all_results.append(lgbm_result)
    print(f"{lgbm_result.name}: accuracy={lgbm_result.accuracy:.3f}")

    for name, pipe, result in train_rf_variants():
        all_results.append(result)
        print(f"{name}: accuracy={result.accuracy:.3f}")

    if save:
        joblib.dump(pipe_rf, ARTIFACTS_DIR / "baseline_rf.joblib")
        joblib.dump(pipe_lgbm, ARTIFACTS_DIR / "baseline_lgbm.joblib")
        with open(ARTIFACTS_DIR / "baseline_results.json", "w", encoding="utf-8") as f:
            json.dump([r.to_dict() for r in all_results], f, indent=2)

    ds = load_baseline_dataset()
    preds = pipe_rf.predict(ds.X_test)
    print("\nBaseline RF classification report:")
    print(format_report(ds.y_test, preds))

    return all_results


if __name__ == "__main__":
    run_baselines()
