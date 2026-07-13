"""Print calibration bins for a saved joblib classifier on a CSV slice."""

from __future__ import annotations

import argparse
from pathlib import Path

import joblib
import numpy as np
import pandas as pd


def positive_proba(model, X: pd.DataFrame) -> np.ndarray:
    if not hasattr(model, "predict_proba"):
        raise SystemExit("model has no predict_proba — need a classifier pipeline")
    proba = model.predict_proba(X)
    if proba.shape[1] == 2:
        return proba[:, 1]
    return proba.max(axis=1)


def calibration_table(y: np.ndarray, p: np.ndarray, bins: int = 10):
    edges = np.linspace(0, 1, bins + 1)
    rows = []
    for i in range(bins):
        lo, hi = edges[i], edges[i + 1]
        mask = (p >= lo) & (p < hi if i < bins - 1 else p <= hi)
        n = int(mask.sum())
        if n == 0:
            continue
        rows.append(
            {
                "bin": f"{lo:.1f}-{hi:.1f}",
                "n": n,
                "pred": float(p[mask].mean()),
                "actual": float(y[mask].mean()),
                "gap": float(abs(p[mask].mean() - y[mask].mean())),
            }
        )
    return rows


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("model", type=Path, help="joblib pipeline path")
    ap.add_argument("csv", type=Path, help="CSV with features + label column")
    ap.add_argument("--target", default="class", help="label column (default: class)")
    ap.add_argument("--positive", default=">50K", help="positive class label for binary metrics")
    ap.add_argument("--bins", type=int, default=10)
    ap.add_argument("--limit", type=int, default=0, help="optional row cap")
    args = ap.parse_args()

    pipe = joblib.load(args.model)
    df = pd.read_csv(args.csv)
    if args.limit:
        df = df.head(args.limit)

    if args.target not in df.columns:
        raise SystemExit(f"column {args.target!r} not in CSV")

    y_raw = df[args.target]
    X = df.drop(columns=[args.target])
    y = (y_raw.astype(str) == args.positive).astype(int).to_numpy()
    p = positive_proba(pipe, X)

    brier = float(np.mean((p - y) ** 2))
    rows = calibration_table(y, p, bins=args.bins)
    ece = sum(r["gap"] * r["n"] for r in rows) / len(y)

    print(f"rows={len(y)}  brier={brier:.4f}  ece={ece:.4f}")
    print(f"{'bin':<10} {'n':>6} {'pred':>7} {'actual':>7} {'gap':>7}")
    for r in rows:
        print(f"{r['bin']:<10} {r['n']:>6} {r['pred']:>7.3f} {r['actual']:>7.3f} {r['gap']:>7.3f}")


if __name__ == "__main__":
    main()
