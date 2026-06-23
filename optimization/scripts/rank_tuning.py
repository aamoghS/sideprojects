"""Run all tuning methods and print a ranked leaderboard."""

from __future__ import annotations

import warnings

warnings.filterwarnings("ignore")

from src.baseline import run_baselines
from src.metrics import ModelResult
from src.tune import REGISTRY, run_method

RF_ITERS = 15
OPTUNA_TRIALS = 20


def main() -> None:
    print("Running baselines...")
    baselines = run_baselines(save=False)
    primary = next(r for r in baselines if r.name == "baseline_rf")

    print("\nRunning all tuning methods (quick settings)...")
    results: list[ModelResult] = []
    skip = {"rf_grid"}  # exhaustive; slow — run separately with: tune --method rf_grid
    for method_id in sorted(REGISTRY.keys()):
        if method_id in skip:
            continue
        method = REGISTRY[method_id]
        print(f"  {method_id}...", flush=True)
        _, result, _ = run_method(method_id, RF_ITERS, OPTUNA_TRIALS)
        results.append(result)

    by_acc = sorted(results, key=lambda r: r.accuracy, reverse=True)
    by_f1 = sorted(results, key=lambda r: r.f1, reverse=True)
    by_auc = sorted(results, key=lambda r: r.roc_auc, reverse=True)

    print("\n" + "=" * 78)
    print(f"Baseline (primary): {primary.name}  accuracy={primary.accuracy:.3f}  f1={primary.f1:.3f}")
    print("=" * 78)

    print("\nRanked by ACCURACY")
    print(f"{'#':<3} {'Method':<28} {'Acc':>7} {'F1':>7} {'AUC':>7} {'Gain':>7}")
    print("-" * 78)
    for i, r in enumerate(by_acc, 1):
        gain = r.accuracy - primary.accuracy
        print(f"{i:<3} {r.name:<28} {r.accuracy:>7.3f} {r.f1:>7.3f} {r.roc_auc:>7.3f} {gain:>+7.3f}")

    print("\nRanked by F1 (better for imbalanced labels)")
    print(f"{'#':<3} {'Method':<28} {'F1':>7} {'Acc':>7} {'Recall':>7}")
    print("-" * 78)
    for i, r in enumerate(by_f1, 1):
        print(f"{i:<3} {r.name:<28} {r.f1:>7.3f} {r.accuracy:>7.3f} {r.recall:>7.3f}")

    print("\nBest overall picks:")
    print(f"  Accuracy:  {by_acc[0].name} ({by_acc[0].accuracy:.3f})")
    print(f"  F1:        {by_f1[0].name} ({by_f1[0].f1:.3f})")
    print(f"  ROC-AUC:   {by_auc[0].name} ({by_auc[0].roc_auc:.3f})")


if __name__ == "__main__":
    main()
