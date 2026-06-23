"""Hyperparameter tuning for Random Forest and LightGBM."""

from __future__ import annotations

import json
from typing import Any

import joblib
import numpy as np
import optuna
from lightgbm import LGBMClassifier, early_stopping
from sklearn.experimental import enable_halving_search_cv  # noqa: F401
from sklearn.ensemble import ExtraTreesClassifier, RandomForestClassifier
from sklearn.metrics import accuracy_score
from sklearn.model_selection import (
    GridSearchCV,
    HalvingRandomSearchCV,
    RandomizedSearchCV,
    cross_val_score,
    train_test_split,
)
from sklearn.pipeline import Pipeline

from src.config import ARTIFACTS_DIR, CV_FOLDS, RANDOM_STATE
from src.data import load_dataset
from src.metrics import ModelResult, evaluate
from src.models import RF_VARIANTS
from src.tuning_catalog import REGISTRY, TuningMethod, register

DEFAULT_METHOD_IDS = (
    "rf_randomized",
    "extra_trees_randomized",
    "lgbm_optuna",
    "rf_optuna_variants",
)


def _fit_transform(ds):
    X_train = ds.preprocessor.fit_transform(ds.X_train)
    X_test = ds.preprocessor.transform(ds.X_test)
    return X_train, X_test


def _evaluate_pipe(pipe: Pipeline, ds, name: str) -> ModelResult:
    preds = pipe.predict(ds.X_test)
    proba = pipe.predict_proba(ds.X_test)[:, 1]
    return evaluate(name, ds.y_test, preds, proba)


def _rf_pipe(clf=None) -> tuple[object, object]:
    ds = load_dataset()
    if clf is None:
        clf = RandomForestClassifier(random_state=RANDOM_STATE, n_jobs=-1)
    pipe = Pipeline([("prep", ds.preprocessor), ("clf", clf)])
    return ds, pipe


RF_PARAM_DIST = {
    "clf__n_estimators": [50, 100, 200, 300, 500],
    "clf__max_depth": [None, 5, 10, 15, 20, 30],
    "clf__min_samples_split": [2, 5, 10, 20],
    "clf__min_samples_leaf": [1, 2, 4, 8],
    "clf__max_features": ["sqrt", "log2", 0.3, 0.5, 0.7],
    "clf__bootstrap": [True, False],
    "clf__class_weight": [None, "balanced"],
}

RF_PARAM_GRID = {
    "clf__n_estimators": [100, 200, 300],
    "clf__max_depth": [10, 20, None],
    "clf__min_samples_split": [2, 5, 10],
    "clf__min_samples_leaf": [1, 2, 4],
    "clf__max_features": ["sqrt", 0.5],
    "clf__class_weight": [None, "balanced"],
}

LGBM_PARAM_DIST = {
    "clf__n_estimators": [50, 100, 200, 300, 500],
    "clf__max_depth": [3, 5, 7, 10, 15],
    "clf__num_leaves": [16, 31, 63, 127],
    "clf__learning_rate": [0.01, 0.03, 0.05, 0.1, 0.2],
    "clf__min_child_samples": [5, 10, 20, 50],
    "clf__subsample": [0.6, 0.8, 1.0],
    "clf__colsample_bytree": [0.6, 0.8, 1.0],
    "clf__reg_alpha": [0.0, 0.1, 1.0],
    "clf__reg_lambda": [0.0, 0.1, 1.0],
    "clf__scale_pos_weight": [1, 2, 3],
}


def tune_rf_randomized(n_iter: int = 40) -> tuple[Pipeline, ModelResult, dict]:
    ds, pipe = _rf_pipe()
    search = RandomizedSearchCV(
        pipe,
        param_distributions=RF_PARAM_DIST,
        n_iter=n_iter,
        cv=CV_FOLDS,
        scoring="accuracy",
        random_state=RANDOM_STATE,
        n_jobs=-1,
        verbose=0,
    )
    search.fit(ds.X_train, ds.y_train)
    best = search.best_estimator_
    result = _evaluate_pipe(best, ds, "rf_randomized")
    return best, result, search.best_params_


def tune_rf_grid() -> tuple[Pipeline, ModelResult, dict]:
    ds, pipe = _rf_pipe()
    search = GridSearchCV(
        pipe,
        param_grid=RF_PARAM_GRID,
        cv=CV_FOLDS,
        scoring="accuracy",
        n_jobs=-1,
        verbose=0,
    )
    search.fit(ds.X_train, ds.y_train)
    best = search.best_estimator_
    result = _evaluate_pipe(best, ds, "rf_grid")
    return best, result, search.best_params_


def tune_rf_halving(n_iter: int = 32) -> tuple[Pipeline, ModelResult, dict]:
    ds, pipe = _rf_pipe()
    search = HalvingRandomSearchCV(
        pipe,
        param_distributions=RF_PARAM_DIST,
        n_candidates=n_iter,
        cv=CV_FOLDS,
        scoring="accuracy",
        random_state=RANDOM_STATE,
        n_jobs=-1,
        verbose=0,
    )
    search.fit(ds.X_train, ds.y_train)
    best = search.best_estimator_
    result = _evaluate_pipe(best, ds, "rf_halving")
    return best, result, search.best_params_


def tune_extra_trees_randomized(n_iter: int = 40) -> tuple[Pipeline, ModelResult, dict]:
    ds = load_dataset()
    pipe = Pipeline(
        [
            ("prep", ds.preprocessor),
            ("clf", ExtraTreesClassifier(random_state=RANDOM_STATE, n_jobs=-1)),
        ]
    )
    search = RandomizedSearchCV(
        pipe,
        param_distributions=RF_PARAM_DIST,
        n_iter=n_iter,
        cv=CV_FOLDS,
        scoring="accuracy",
        random_state=RANDOM_STATE,
        n_jobs=-1,
        verbose=0,
    )
    search.fit(ds.X_train, ds.y_train)
    best = search.best_estimator_
    result = _evaluate_pipe(best, ds, "extra_trees_randomized")
    return best, result, search.best_params_


def tune_lgbm_randomized(n_iter: int = 40) -> tuple[Pipeline, ModelResult, dict]:
    ds = load_dataset()
    pipe = Pipeline(
        [
            ("prep", ds.preprocessor),
            ("clf", LGBMClassifier(random_state=RANDOM_STATE, n_jobs=-1, verbose=-1)),
        ]
    )
    search = RandomizedSearchCV(
        pipe,
        param_distributions=LGBM_PARAM_DIST,
        n_iter=n_iter,
        cv=CV_FOLDS,
        scoring="accuracy",
        random_state=RANDOM_STATE,
        n_jobs=-1,
        verbose=0,
    )
    search.fit(ds.X_train, ds.y_train)
    best = search.best_estimator_
    result = _evaluate_pipe(best, ds, "lgbm_randomized")
    return best, result, search.best_params_


def tune_lgbm_optuna(n_trials: int = 50) -> tuple[Pipeline, ModelResult, dict]:
    ds = load_dataset()
    X_train, _ = _fit_transform(ds)
    y_train = ds.y_train

    def objective(trial: optuna.Trial) -> float:
        params = {
            "n_estimators": trial.suggest_int("n_estimators", 50, 500),
            "max_depth": trial.suggest_int("max_depth", 3, 15),
            "num_leaves": trial.suggest_int("num_leaves", 8, 128),
            "learning_rate": trial.suggest_float("learning_rate", 0.01, 0.3, log=True),
            "min_child_samples": trial.suggest_int("min_child_samples", 5, 100),
            "subsample": trial.suggest_float("subsample", 0.5, 1.0),
            "colsample_bytree": trial.suggest_float("colsample_bytree", 0.5, 1.0),
            "reg_alpha": trial.suggest_float("reg_alpha", 1e-8, 10.0, log=True),
            "reg_lambda": trial.suggest_float("reg_lambda", 1e-8, 10.0, log=True),
        }
        clf = LGBMClassifier(**params, random_state=RANDOM_STATE, n_jobs=-1, verbose=-1)
        scores = cross_val_score(clf, X_train, y_train, cv=CV_FOLDS, scoring="accuracy", n_jobs=-1)
        return float(np.mean(scores))

    optuna.logging.set_verbosity(optuna.logging.WARNING)
    study = optuna.create_study(direction="maximize", sampler=optuna.samplers.TPESampler(seed=RANDOM_STATE))
    study.optimize(objective, n_trials=n_trials, show_progress_bar=False)

    best_params = study.best_params
    clf = LGBMClassifier(**best_params, random_state=RANDOM_STATE, n_jobs=-1, verbose=-1)
    pipe = Pipeline([("prep", ds.preprocessor), ("clf", clf)])
    pipe.fit(ds.X_train, ds.y_train)
    result = _evaluate_pipe(pipe, ds, "lgbm_optuna")
    return pipe, result, best_params


def tune_lgbm_optuna_pruned(n_trials: int = 50) -> tuple[Pipeline, ModelResult, dict]:
    """Optuna with early stopping on a validation split — stops boosting when loss plateaus."""
    ds = load_dataset()
    X_train, _ = _fit_transform(ds)
    y_train = ds.y_train
    X_tr, X_val, y_tr, y_val = train_test_split(
        X_train, y_train, test_size=0.2, random_state=RANDOM_STATE, stratify=y_train
    )

    def objective(trial: optuna.Trial) -> float:
        params = {
            "n_estimators": trial.suggest_int("n_estimators", 100, 800),
            "max_depth": trial.suggest_int("max_depth", 3, 15),
            "num_leaves": trial.suggest_int("num_leaves", 8, 128),
            "learning_rate": trial.suggest_float("learning_rate", 0.01, 0.3, log=True),
            "min_child_samples": trial.suggest_int("min_child_samples", 5, 100),
            "subsample": trial.suggest_float("subsample", 0.5, 1.0),
            "colsample_bytree": trial.suggest_float("colsample_bytree", 0.5, 1.0),
        }
        clf = LGBMClassifier(**params, random_state=RANDOM_STATE, n_jobs=-1, verbose=-1)
        clf.fit(
            X_tr,
            y_tr,
            eval_set=[(X_val, y_val)],
            eval_metric="binary_logloss",
            callbacks=[early_stopping(30, verbose=False)],
        )
        preds = clf.predict(X_val)
        return accuracy_score(y_val, preds)

    optuna.logging.set_verbosity(optuna.logging.WARNING)
    study = optuna.create_study(
        direction="maximize",
        sampler=optuna.samplers.TPESampler(seed=RANDOM_STATE),
        pruner=optuna.pruners.MedianPruner(n_startup_trials=5),
    )
    study.optimize(objective, n_trials=n_trials, show_progress_bar=False)

    best_params = study.best_params
    clf = LGBMClassifier(**best_params, random_state=RANDOM_STATE, n_jobs=-1, verbose=-1)
    pipe = Pipeline([("prep", ds.preprocessor), ("clf", clf)])
    pipe.fit(ds.X_train, ds.y_train)
    result = _evaluate_pipe(pipe, ds, "lgbm_optuna_pruned")
    return pipe, result, best_params


def tune_rf_optuna_variants(n_trials: int = 50) -> tuple[Pipeline, ModelResult, dict]:
    ds = load_dataset()
    X_train, _ = _fit_transform(ds)
    y_train = ds.y_train

    def objective(trial: optuna.Trial) -> float:
        variant = trial.suggest_categorical("variant", list(RF_VARIANTS.keys()))
        params: dict[str, Any] = {
            "n_estimators": trial.suggest_int("n_estimators", 50, 500),
            "max_depth": trial.suggest_int("max_depth", 3, 30),
            "min_samples_split": trial.suggest_int("min_samples_split", 2, 30),
            "min_samples_leaf": trial.suggest_int("min_samples_leaf", 1, 20),
            "max_features": trial.suggest_categorical("max_features", ["sqrt", "log2", 0.5]),
        }
        if variant == "balanced_rf":
            params["class_weight"] = "balanced"
        if variant == "bootstrap_false_rf":
            params["bootstrap"] = False

        clf = RF_VARIANTS[variant](**params)
        scores = cross_val_score(clf, X_train, y_train, cv=CV_FOLDS, scoring="accuracy", n_jobs=-1)
        return float(np.mean(scores))

    optuna.logging.set_verbosity(optuna.logging.WARNING)
    study = optuna.create_study(direction="maximize", sampler=optuna.samplers.TPESampler(seed=RANDOM_STATE))
    study.optimize(objective, n_trials=n_trials, show_progress_bar=False)

    best = study.best_params.copy()
    variant = best.pop("variant")
    clf = RF_VARIANTS[variant](**best)
    pipe = Pipeline([("prep", ds.preprocessor), ("clf", clf)])
    pipe.fit(ds.X_train, ds.y_train)
    result = _evaluate_pipe(pipe, ds, f"rf_optuna_{variant}")
    best["variant"] = variant
    return pipe, result, best


def tune_threshold_on_rf(n_iter: int = 40) -> tuple[Pipeline, ModelResult, dict]:
    """Train a tuned RF, then optimize the decision threshold (default is 0.5)."""
    pipe, _, rf_params = tune_rf_randomized(n_iter)
    ds = load_dataset()
    X_train, _ = _fit_transform(ds)
    y_train = ds.y_train

    proba = pipe.predict_proba(ds.X_test)[:, 1]
    train_proba = pipe.predict_proba(ds.X_train)[:, 1]

    best_threshold = 0.5
    best_acc = 0.0
    for threshold in np.arange(0.25, 0.75, 0.01):
        preds = (train_proba >= threshold).astype(int)
        acc = accuracy_score(y_train, preds)
        if acc > best_acc:
            best_acc = acc
            best_threshold = float(threshold)

    test_preds = (proba >= best_threshold).astype(int)
    result = evaluate("rf_threshold_tuned", ds.y_test, test_preds, proba)
    params = {**rf_params, "decision_threshold": best_threshold}
    return pipe, result, params


def tune_lgbm_f1_optuna(n_trials: int = 50) -> tuple[Pipeline, ModelResult, dict]:
    """Same as Optuna LGBM but optimizes F1 instead of accuracy — better for imbalanced data."""
    ds = load_dataset()
    X_train, _ = _fit_transform(ds)
    y_train = ds.y_train

    def objective(trial: optuna.Trial) -> float:
        params = {
            "n_estimators": trial.suggest_int("n_estimators", 50, 500),
            "max_depth": trial.suggest_int("max_depth", 3, 15),
            "num_leaves": trial.suggest_int("num_leaves", 8, 128),
            "learning_rate": trial.suggest_float("learning_rate", 0.01, 0.3, log=True),
            "scale_pos_weight": trial.suggest_float("scale_pos_weight", 1.0, 5.0),
        }
        clf = LGBMClassifier(**params, random_state=RANDOM_STATE, n_jobs=-1, verbose=-1)
        scores = cross_val_score(clf, X_train, y_train, cv=CV_FOLDS, scoring="f1", n_jobs=-1)
        return float(np.mean(scores))

    optuna.logging.set_verbosity(optuna.logging.WARNING)
    study = optuna.create_study(direction="maximize", sampler=optuna.samplers.TPESampler(seed=RANDOM_STATE))
    study.optimize(objective, n_trials=n_trials, show_progress_bar=False)

    best_params = study.best_params
    clf = LGBMClassifier(**best_params, random_state=RANDOM_STATE, n_jobs=-1, verbose=-1)
    pipe = Pipeline([("prep", ds.preprocessor), ("clf", clf)])
    pipe.fit(ds.X_train, ds.y_train)
    result = _evaluate_pipe(pipe, ds, "lgbm_f1_optuna")
    return pipe, result, best_params


def _register_all() -> None:
    register(
        TuningMethod(
            id="rf_randomized",
            name="Randomized Search",
            category="1. Search strategies",
            model="Random Forest",
            description="Sample random hyperparameter combos; cheap way to explore a large space.",
            why_it_helps="Finds good tree depth, n_estimators, and feature sampling without exhaustive grid cost.",
            runner=tune_rf_randomized,
        )
    )
    register(
        TuningMethod(
            id="rf_grid",
            name="Grid Search",
            category="1. Search strategies",
            model="Random Forest",
            description="Try every combo in a focused grid - exhaustive but slow.",
            why_it_helps="Guarantees the best combo within the grid; good when you already know the rough sweet spot.",
            runner=tune_rf_grid,
        )
    )
    register(
        TuningMethod(
            id="rf_halving",
            name="Successive Halving (HalvingRandomSearchCV)",
            category="1. Search strategies",
            model="Random Forest",
            description="Train many configs on small data slices; promote only the top performers to full data.",
            why_it_helps="Same quality as random search in less time - kills weak configs early.",
            runner=tune_rf_halving,
        )
    )
    register(
        TuningMethod(
            id="extra_trees_randomized",
            name="Randomized Search",
            category="2. Model variants",
            model="Extra Trees",
            description="Same search as RF but on Extremely Randomized Trees (random split points).",
            why_it_helps="Extra Trees often generalize better with less overfitting than standard RF.",
            runner=tune_extra_trees_randomized,
        )
    )
    register(
        TuningMethod(
            id="rf_optuna_variants",
            name="Optuna TPE + variant picker",
            category="2. Model variants",
            model="Random Forest family",
            description="Bayesian-style search that also picks the best RF type (standard, balanced, bootstrap-off).",
            why_it_helps="Lets the data choose between RF, balanced RF, extra trees, and bootstrap-off in one run.",
            runner=tune_rf_optuna_variants,
        )
    )
    register(
        TuningMethod(
            id="lgbm_randomized",
            name="Randomized Search",
            category="3. LightGBM",
            model="LightGBM",
            description="Random search over learning rate, leaves, subsample, and regularization.",
            why_it_helps="LightGBM has many interacting params; random search finds strong combos fast.",
            runner=tune_lgbm_randomized,
        )
    )
    register(
        TuningMethod(
            id="lgbm_optuna",
            name="Optuna TPE",
            category="3. LightGBM",
            model="LightGBM",
            description="Tree-structured Parzen Estimator — learns from past trials to pick smarter next params.",
            why_it_helps="Usually beats random search with the same trial budget; great for gradient boosting.",
            runner=tune_lgbm_optuna,
        )
    )
    register(
        TuningMethod(
            id="lgbm_optuna_pruned",
            name="Optuna TPE + early stopping",
            category="3. LightGBM",
            model="LightGBM",
            description="Optuna search with LightGBM early stopping on a validation split.",
            why_it_helps="Finds the right number of boosting rounds automatically; avoids overfitting from too many trees.",
            runner=tune_lgbm_optuna_pruned,
        )
    )
    register(
        TuningMethod(
            id="lgbm_f1_optuna",
            name="Optuna TPE (F1 objective)",
            category="4. Metric-aware tuning",
            model="LightGBM",
            description="Optimizes F1 during search instead of accuracy, plus scale_pos_weight for imbalance.",
            why_it_helps="Adult dataset is imbalanced (~24% >50K); accuracy alone hides poor minority-class recall.",
            runner=tune_lgbm_f1_optuna,
        )
    )
    register(
        TuningMethod(
            id="rf_threshold",
            name="Decision threshold tuning",
            category="4. Metric-aware tuning",
            model="Random Forest",
            description="After training, sweep the probability cutoff (default 0.5) on train set.",
            why_it_helps="Free accuracy/F1 boost without retraining — shifts the precision/recall tradeoff.",
            runner=tune_threshold_on_rf,
        )
    )


_register_all()


def run_method(method_id: str, n_iter: int = 40, n_trials: int = 50) -> tuple[Pipeline, ModelResult, dict]:
    method = REGISTRY[method_id]
    if method_id in ("rf_grid",):
        return method.runner()
    if method_id in ("rf_threshold",):
        return method.runner(n_iter)
    if method_id in ("rf_randomized", "rf_halving", "extra_trees_randomized", "lgbm_randomized"):
        return method.runner(n_iter)
    return method.runner(n_trials)


def run_tuning(
    rf_randomized_iters: int = 40,
    optuna_trials: int = 50,
    save: bool = True,
    method_ids: tuple[str, ...] | None = None,
) -> list[ModelResult]:
    ARTIFACTS_DIR.mkdir(parents=True, exist_ok=True)
    results: list[ModelResult] = []
    params_log: dict[str, dict] = {}
    ids = method_ids or DEFAULT_METHOD_IDS

    for method_id in ids:
        method = REGISTRY[method_id]
        print(f"\n--- {method.category}: {method.name} ({method.model}) ---")
        pipe, result, params = run_method(method_id, rf_randomized_iters, optuna_trials)
        results.append(result)
        params_log[result.name] = params
        print(f"{result.name}: accuracy={result.accuracy:.3f}  f1={result.f1:.3f}")

        if save:
            joblib.dump(pipe, ARTIFACTS_DIR / f"{result.name}.joblib")

    if save:
        with open(ARTIFACTS_DIR / "tuned_results.json", "w", encoding="utf-8") as f:
            json.dump([r.to_dict() for r in results], f, indent=2)
        with open(ARTIFACTS_DIR / "best_params.json", "w", encoding="utf-8") as f:
            json.dump(params_log, f, indent=2)

    return results


if __name__ == "__main__":
    run_tuning()
