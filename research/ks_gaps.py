"""Two-sample KS distance between two numeric CSV columns."""

from __future__ import annotations

import argparse
from pathlib import Path

import numpy as np
import pandas as pd


def ks_distance(a: np.ndarray, b: np.ndarray) -> float:
    a = np.sort(a[np.isfinite(a)])
    b = np.sort(b[np.isfinite(b)])
    if len(a) == 0 or len(b) == 0:
        raise ValueError("need finite values in both columns")

    data = np.sort(np.concatenate([a, b]))
    cdf_a = np.searchsorted(a, data, side="right") / len(a)
    cdf_b = np.searchsorted(b, data, side="right") / len(b)
    return float(np.max(np.abs(cdf_a - cdf_b)))


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("csv", type=Path)
    ap.add_argument("col_a", help="first column")
    ap.add_argument("col_b", help="second column")
    ap.add_argument("--where", default="", help="pandas query filter, e.g. isFraud == 0")
    ap.add_argument("--sample", type=int, default=0, help="random subsample size")
    args = ap.parse_args()

    df = pd.read_csv(args.csv)
    if args.where:
        df = df.query(args.where)

    if args.sample and len(df) > args.sample:
        df = df.sample(args.sample, random_state=0)

    a = pd.to_numeric(df[args.col_a], errors="coerce").to_numpy()
    b = pd.to_numeric(df[args.col_b], errors="coerce").to_numpy()

    d = ks_distance(a, b)
    print(f"file: {args.csv}")
    if args.where:
        print(f"filter: {args.where}")
    print(f"cols: {args.col_a} (n={np.isfinite(a).sum()}) vs {args.col_b} (n={np.isfinite(b).sum()})")
    print(f"KS D = {d:.4f}")

    if d < 0.05:
        print("looks close - distributions barely separate at this sample size")
    elif d < 0.15:
        print("mild shift - worth plotting hist/CDF if you care")
    else:
        print("big gap - these columns do not look like the same underlying distribution")


if __name__ == "__main__":
    main()
