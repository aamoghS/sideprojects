"""Compare baseline vs tuned models and plot accuracy gains."""

from __future__ import annotations

import json
from pathlib import Path

import matplotlib.pyplot as plt

from src.baseline import run_baselines
from src.config import ARTIFACTS_DIR
from src.metrics import ModelResult
from src.tune import run_tuning


def _load_json(path: Path) -> list[dict]:
    if not path.exists():
        return []
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def print_comparison(baseline: list[ModelResult], tuned: list[ModelResult]) -> None:
    print("\n" + "=" * 72)
    print(f"{'Model':<30} {'Accuracy':>10} {'F1':>10} {'ROC-AUC':>10}")
    print("-" * 72)

    print("\nBaselines (weak hyperparameters, ~60% target):")
    for r in baseline:
        print(f"  {r.name:<28} {r.accuracy:>10.3f} {r.f1:>10.3f} {r.roc_auc:>10.3f}")

    print("\nTuned models:")
    for r in tuned:
        print(f"  {r.name:<28} {r.accuracy:>10.3f} {r.f1:>10.3f} {r.roc_auc:>10.3f}")

    if baseline and tuned:
        primary_base = next((r for r in baseline if r.name == "baseline_rf"), baseline[0])
        best_tuned = max(tuned, key=lambda x: x.accuracy)
        gain = best_tuned.accuracy - primary_base.accuracy
        print("\n" + "-" * 72)
        print(f"Primary baseline: {primary_base.name} ({primary_base.accuracy:.3f})")
        print(f"Best tuned:       {best_tuned.name} ({best_tuned.accuracy:.3f})")
        print(f"Accuracy gain:    +{gain:.3f} ({gain / primary_base.accuracy * 100:.1f}% relative)")


def plot_comparison(baseline: list[ModelResult], tuned: list[ModelResult]) -> Path:
    ARTIFACTS_DIR.mkdir(parents=True, exist_ok=True)
    names = [r.name for r in baseline + tuned]
    accs = [r.accuracy for r in baseline + tuned]
    colors = ["#94a3b8"] * len(baseline) + ["#22c55e"] * len(tuned)

    fig, ax = plt.subplots(figsize=(12, 5))
    bars = ax.bar(names, accs, color=colors)
    ax.axhline(0.60, color="#ef4444", linestyle="--", linewidth=1, label="60% target baseline")
    ax.set_ylabel("Accuracy")
    ax.set_title("Baseline vs Tuned Model Accuracy")
    ax.set_ylim(0.5, 1.0)
    ax.tick_params(axis="x", rotation=45)
    plt.setp(ax.get_xticklabels(), ha="right")
    for bar, acc in zip(bars, accs):
        ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height() + 0.005, f"{acc:.3f}", ha="center", va="bottom", fontsize=8)
    ax.legend()
    fig.tight_layout()
    out = ARTIFACTS_DIR / "accuracy_comparison.png"
    fig.savefig(out, dpi=150)
    plt.close(fig)
    return out


def run_full_pipeline(
    rf_randomized_iters: int = 40,
    optuna_trials: int = 50,
    skip_tune: bool = False,
) -> None:
    print("=== Phase 1: Baselines ===")
    baseline = run_baselines()

    if skip_tune:
        tuned: list[ModelResult] = []
    else:
        print("\n=== Phase 2: Hyperparameter tuning ===")
        tuned = run_tuning(rf_randomized_iters, optuna_trials)

    print_comparison(baseline, tuned)
    if baseline or tuned:
        chart = plot_comparison(baseline, tuned)
        print(f"\nChart saved to {chart}")


def compare_saved() -> None:
    baseline_raw = _load_json(ARTIFACTS_DIR / "baseline_results.json")
    tuned_raw = _load_json(ARTIFACTS_DIR / "tuned_results.json")
    baseline = [ModelResult(**d) for d in baseline_raw]
    tuned = [ModelResult(**d) for d in tuned_raw]
    print_comparison(baseline, tuned)
    if baseline or tuned:
        chart = plot_comparison(baseline, tuned)
        print(f"\nChart saved to {chart}")


if __name__ == "__main__":
    run_full_pipeline()
