"""
Runs the adaptive retraining pipeline across multiple dataset types and
generates a comparative multi-line graph showing performance across all of them.
"""
import os
import numpy as np
import pandas as pd
import joblib
import lightgbm as lgb
from sklearn.preprocessing import LabelEncoder
from sklearn.metrics import roc_auc_score
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import matplotlib.gridspec as gridspec
import seaborn as sns
import logging

from retrain_pipeline import AdaptiveRetrainer

logging.getLogger('retrain_pipeline').setLevel(logging.CRITICAL)
logging.getLogger('lightgbm').setLevel(logging.CRITICAL)

CHUNK_SIZE   = 1000
WINDOW_SIZE  = 12000
OUT_PNG      = r'C:\Users\bootcamp\Desktop\sideprojects\research\benchmark_comparison.png'

DATASETS = [
    {
        'name': 'PaySim (Banking)',
        'path': 'paysim.csv',
        'color': '#4C9BE8',
        'description': 'Mobile money fraud\n(original baseline)',
    },
    {
        'name': 'E-Commerce',
        'path': 'dataset_ecommerce.csv',
        'color': '#F4845F',
        'description': 'Chargeback & refund fraud\n(mid-year drift)',
    },
    {
        'name': 'Crypto Exchange',
        'path': 'dataset_crypto.csv',
        'color': '#A8D5A2',
        'description': 'Wash trading & pump-dump\n(heavy-tail, volatile)',
    },
    {
        'name': 'Insurance Claims',
        'path': 'dataset_insurance.csv',
        'color': '#C77DFF',
        'description': 'Claim ring fraud\n(very imbalanced, 2 drift waves)',
    },
]


def bootstrap_models(df: pd.DataFrame, nrows: int = 12000):
    """Train initial LightGBM + IsolationForest on first nrows."""
    from sklearn.ensemble import IsolationForest

    le = LabelEncoder()
    df = df.copy()
    df['type'] = le.fit_transform(df['type'].astype(str))
    joblib.dump(le, 'label_encoder.joblib')

    train_df = df.iloc[:nrows].copy()
    features = ['step', 'type', 'amount', 'oldbalanceOrg', 'newbalanceOrig',
                'oldbalanceDest', 'newbalanceDest']
    train_df['balance_diff'] = train_df['newbalanceOrig'] - train_df['oldbalanceOrg']
    train_df['amount_to_oldbalance_ratio'] = train_df['amount'] / (train_df['oldbalanceOrg'] + 1)
    train_df['dest_balance_diff'] = train_df['newbalanceDest'] - train_df['oldbalanceDest']
    all_feats = features + ['balance_diff', 'amount_to_oldbalance_ratio', 'dest_balance_diff']

    params = {'objective': 'binary', 'metric': 'auc', 'learning_rate': 0.05,
              'num_leaves': 15, 'max_depth': 4, 'verbose': -1}
    lgb_data = lgb.Dataset(train_df[all_feats], label=train_df['isFraud'])
    model = lgb.train(params, lgb_data, num_boost_round=100)
    model.save_model('primary_model.txt')

    iso = IsolationForest(n_estimators=100, contamination=0.05, random_state=42)
    iso.fit(train_df[all_feats])
    joblib.dump(iso, 'primary_if.joblib')


def run_dataset(ds: dict) -> dict:
    """Run the full 12-month adaptive pipeline on one dataset. Returns tick AUC trace."""
    name = ds['name']
    path = ds['path']
    print(f"\n{'='*60}")
    print(f"  Dataset: {name}")
    print(f"{'='*60}")

    if not os.path.exists(path):
        print(f"  [SKIP] File not found: {path}")
        return {'name': name, 'ticks': [], 'p_aucs': [], 'color': ds['color']}

    df = pd.read_csv(path).sort_values('step').reset_index(drop=True)
    print(f"  Rows: {len(df):,}  |  Fraud rate: {df['isFraud'].mean()*100:.2f}%")

    bootstrap_models(df)

    retrainer = AdaptiveRetrainer(
        data_path='temp_bench.csv',
        primary_path='primary_model.txt',
        candidate_path='candidate_model.txt',
        encoder_path='label_encoder.joblib',
    )

    ticks, p_aucs = [], []
    total_ticks = 52000 // CHUNK_SIZE
    promotions = 0

    for tick in range(1, total_ticks + 1):
        start_idx = (tick - 1) * CHUNK_SIZE
        end_idx   = start_idx + WINDOW_SIZE + CHUNK_SIZE
        if end_idx > len(df):
            break
        chunk = df.iloc[start_idx:end_idx].copy()
        chunk.to_csv('temp_bench.csv', index=False)

        p_auc, c_auc, promoted = retrainer.execute_pipeline(
            initial_window=WINDOW_SIZE, test_size=CHUNK_SIZE
        )

        ticks.append(tick)
        if p_auc is not None:
            p_aucs.append(p_auc)
            regime = 'HEALTHY' if c_auc is None else ('WATCH' if p_auc >= 0.90 else 'DRIFT')
            status = 'WIN' if promoted else ('STATIC' if c_auc is None else 'HOLD')
            c_str = f"-> C:{c_auc:.3f}" if c_auc is not None else ""
            print(f"  Tk {tick:03d} | [{regime}] {status} P:{p_auc:.3f} {c_str}")
            if promoted:
                promotions += 1
        else:
            p_aucs.append(p_aucs[-1] if p_aucs else 0.95)
            print(f"  Tk {tick:03d} | SKIP")

        if os.path.exists('temp_bench.csv'):
            os.remove('temp_bench.csv')

    print(f"  Total promotions: {promotions}")
    return {'name': name, 'ticks': ticks, 'p_aucs': p_aucs,
            'color': ds['color'], 'description': ds['description'],
            'promotions': promotions}


def plot_comparison(results: list):
    sns.set_theme(style='darkgrid')
    fig = plt.figure(figsize=(18, 12))
    fig.patch.set_facecolor('#0F1117')

    gs = gridspec.GridSpec(2, 2, figure=fig, hspace=0.45, wspace=0.35)

    axs = [fig.add_subplot(gs[i // 2, i % 2]) for i in range(4)]

    for ax, res in zip(axs, results):
        if not res['ticks']:
            ax.text(0.5, 0.5, 'Dataset not found', ha='center', va='center',
                    transform=ax.transAxes, color='#888')
            ax.set_facecolor('#1A1D27')
            continue

        ticks  = res['ticks']
        p_aucs = res['p_aucs']
        color  = res['color']

        ax.set_facecolor('#1A1D27')
        ax.tick_params(colors='#CCCCCC')
        for spine in ax.spines.values():
            spine.set_edgecolor('#333')

        # Smooth line (rolling average over 3 ticks)
        smooth = pd.Series(p_aucs).rolling(3, min_periods=1, center=True).mean().tolist()

        ax.plot(ticks, p_aucs, color=color, linewidth=1.0, alpha=0.30)
        ax.plot(ticks, smooth, color=color, linewidth=2.5, label='Production AUC (smoothed)')

        # Floor lines
        ax.axhline(0.95, color='#FFD700', linewidth=1.0, linestyle='--', alpha=0.6, label='HEALTHY floor (0.95)')
        ax.axhline(0.90, color='#FF6B6B', linewidth=1.0, linestyle=':', alpha=0.6, label='DRIFT floor (0.90)')

        ax.set_ylim(0.3, 1.05)
        ax.set_xlim(1, max(ticks) + 1)
        ax.set_title(f"{res['name']}", fontsize=13, fontweight='bold',
                     color='#FFFFFF', pad=10)
        ax.set_xlabel('Week in Production', fontsize=10, color='#AAAAAA')
        ax.set_ylabel('ROC AUC', fontsize=10, color='#AAAAAA')
        ax.legend(fontsize=8, loc='lower right',
                  facecolor='#1A1D27', labelcolor='#CCCCCC', framealpha=0.7)

        # Min AUC annotation
        min_auc = min(p_aucs)
        min_tick = ticks[p_aucs.index(min_auc)]
        ax.annotate(f'Min: {min_auc:.3f}',
                    xy=(min_tick, min_auc), xytext=(min_tick + 1, min_auc + 0.05),
                    fontsize=8, color='#FF6B6B',
                    arrowprops=dict(arrowstyle='->', color='#FF6B6B', lw=1.2))

    fig.suptitle('Adaptive Retraining Pipeline — Multi-Dataset Benchmark\n(Quant-Grade Three-Tier Drift Management)',
                 fontsize=16, fontweight='bold', color='#FFFFFF', y=0.98)

    plt.savefig(OUT_PNG, dpi=200, bbox_inches='tight', facecolor=fig.get_facecolor())
    print(f"\nComparison graph saved -> {OUT_PNG}")


if __name__ == '__main__':
    all_results = []
    for ds in DATASETS:
        result = run_dataset(ds)
        all_results.append(result)

    plot_comparison(all_results)
    print("\nBenchmark complete.")
