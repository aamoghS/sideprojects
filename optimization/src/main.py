"""CLI entry point for the RF / LightGBM optimization lab."""

from __future__ import annotations

import argparse

from src.baseline import run_baselines
from src.compare import compare_saved, run_full_pipeline
from src.tune import DEFAULT_METHOD_IDS, REGISTRY, run_method, run_tuning
from src.tuning_catalog import print_catalog


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Random Forest & LightGBM optimization lab (Adult income dataset)"
    )
    sub = parser.add_subparsers(dest="command", required=True)

    sub.add_parser("baseline", help="Train weak baseline models (~60% accuracy)")
    sub.add_parser("methods", help="List all tuning methods and what each one does")

    tune_p = sub.add_parser("tune", help="Hyperparameter tuning")
    tune_p.add_argument("--method", choices=sorted(REGISTRY.keys()), help="Run a single tuning method")
    tune_p.add_argument("--all-methods", action="store_true", help="Run every tuning method")
    tune_p.add_argument("--rf-iters", type=int, default=40, help="RandomizedSearch / halving iterations")
    tune_p.add_argument("--optuna-trials", type=int, default=50, help="Optuna trials per study")

    all_p = sub.add_parser("all", help="Run baseline + default tuning + comparison")
    all_p.add_argument("--rf-iters", type=int, default=40)
    all_p.add_argument("--optuna-trials", type=int, default=50)
    all_p.add_argument("--quick", action="store_true", help="Fewer search iterations for a fast run")

    sub.add_parser("compare", help="Re-print comparison from saved artifacts")

    args = parser.parse_args()

    if args.command == "baseline":
        run_baselines()
    elif args.command == "methods":
        print_catalog()
    elif args.command == "tune":
        if args.method:
            _, result, params = run_method(args.method, args.rf_iters, args.optuna_trials)
            print(f"\n{result.name}: accuracy={result.accuracy:.3f}  f1={result.f1:.3f}")
            print(f"Best params: {params}")
        elif args.all_methods:
            run_tuning(args.rf_iters, args.optuna_trials, method_ids=tuple(sorted(REGISTRY.keys())))
        else:
            run_tuning(args.rf_iters, args.optuna_trials, method_ids=DEFAULT_METHOD_IDS)
    elif args.command == "all":
        rf_iters = 15 if args.quick else args.rf_iters
        optuna_trials = 20 if args.quick else args.optuna_trials
        run_full_pipeline(rf_iters, optuna_trials)
    elif args.command == "compare":
        compare_saved()


if __name__ == "__main__":
    main()
