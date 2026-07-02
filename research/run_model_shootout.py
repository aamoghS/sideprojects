"""
Model Shootout: Tests every major ML model type through the same
adaptive three-tier quant pipeline on all four datasets.
Shows a multi-line comparison graph per dataset.
"""
import os
import warnings
import numpy as np
import pandas as pd
import joblib
import lightgbm as lgb
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import matplotlib.gridspec as gridspec
import seaborn as sns
from sklearn.preprocessing import LabelEncoder
from sklearn.metrics import roc_auc_score
from sklearn.ensemble import (
    RandomForestClassifier,
    GradientBoostingClassifier,
    ExtraTreesClassifier,
    IsolationForest,
)
from sklearn.linear_model import LogisticRegression

warnings.filterwarnings('ignore')

# ─── Dataset configs ─────────────────────────────────────────────────────────
DATASETS = [
    {'name': 'PaySim (Banking)',    'path': 'paysim.csv',             'color_idx': 0},
    {'name': 'E-Commerce',          'path': 'dataset_ecommerce.csv',  'color_idx': 1},
    {'name': 'Crypto Exchange',     'path': 'dataset_crypto.csv',     'color_idx': 2},
    {'name': 'Insurance Claims',    'path': 'dataset_insurance.csv',  'color_idx': 3},
]

CHUNK_SIZE  = 1000
WINDOW_SIZE = 12000
FEATURES    = ['step', 'type', 'amount', 'oldbalanceOrg', 'newbalanceOrig',
               'oldbalanceDest', 'newbalanceDest',
               'balance_diff', 'amount_to_oldbalance_ratio', 'dest_balance_diff']
HEALTHY_FLOOR = 0.95
DRIFT_FLOOR   = 0.90
OUT_PNG = r'C:\Users\bootcamp\Desktop\sideprojects\research\model_shootout.png'


# ─── Model registry ──────────────────────────────────────────────────────────
def get_model_registry():
    registry = {
        'LightGBM': {
            'color': '#4C9BE8',
            'ls': '-',
            'lw': 2.5,
        },
        'XGBoost': {
            'color': '#F4845F',
            'ls': '-',
            'lw': 2.0,
        },
        'CatBoost': {
            'color': '#A8D5A2',
            'ls': '-',
            'lw': 2.0,
        },
        'Random Forest': {
            'color': '#C77DFF',
            'ls': '--',
            'lw': 2.0,
        },
        'Extra Trees': {
            'color': '#FFD166',
            'ls': '--',
            'lw': 2.0,
        },
        'Gradient Boost': {
            'color': '#EF476F',
            'ls': ':',
            'lw': 2.0,
        },
        'Logistic Reg': {
            'color': '#888888',
            'ls': ':',
            'lw': 1.5,
        },
    }
    # Conditionally add XGBoost/CatBoost
    try:
        import xgboost
    except ImportError:
        del registry['XGBoost']
    try:
        import catboost
    except ImportError:
        del registry['CatBoost']
    return registry


# ─── Feature engineering ─────────────────────────────────────────────────────
def engineer_features(df: pd.DataFrame) -> pd.DataFrame:
    df = df.copy()
    df['balance_diff'] = df['newbalanceOrig'] - df['oldbalanceOrg']
    df['amount_to_oldbalance_ratio'] = df['amount'] / (df['oldbalanceOrg'] + 1)
    df['dest_balance_diff'] = df['newbalanceDest'] - df['oldbalanceDest']
    return df


def time_decay_weights(n: int, window_size: int) -> np.ndarray:
    decay_rate = np.log(2) / (window_size / 2)
    ages = np.arange(n)[::-1]
    return np.exp(-decay_rate * ages)


# ─── Per-model train / predict ────────────────────────────────────────────────
def train_model(name: str, X_train: pd.DataFrame, y_train: pd.Series,
                weights: np.ndarray, primary_path: str) -> object:
    if name == 'LightGBM':
        train_data = lgb.Dataset(X_train, label=y_train, weight=weights)
        params = {'objective': 'binary', 'metric': 'auc', 'learning_rate': 0.05,
                  'num_leaves': 31, 'max_depth': 6, 'verbose': -1,
                  'min_child_samples': 20, 'subsample': 0.8, 'colsample_bytree': 0.8}
        use_init = os.path.exists(primary_path)
        return lgb.train(params, train_data, num_boost_round=100,
                         init_model=primary_path if use_init else None)

    elif name == 'XGBoost':
        import xgboost as xgb
        scale_pos = max(1, int((y_train == 0).sum() / max(1, (y_train == 1).sum())))
        clf = xgb.XGBClassifier(
            n_estimators=100, max_depth=6, learning_rate=0.05,
            subsample=0.8, colsample_bytree=0.8,
            scale_pos_weight=scale_pos, eval_metric='auc',
            use_label_encoder=False, verbosity=0, random_state=42)
        clf.fit(X_train, y_train, sample_weight=weights)
        return clf

    elif name == 'CatBoost':
        from catboost import CatBoostClassifier
        clf = CatBoostClassifier(
            iterations=100, depth=6, learning_rate=0.05,
            eval_metric='AUC', verbose=False, random_seed=42,
            auto_class_weights='Balanced')
        clf.fit(X_train, y_train, sample_weight=weights)
        return clf

    elif name == 'Random Forest':
        n_pos = max(1, (y_train == 1).sum())
        n_neg = max(1, (y_train == 0).sum())
        clf = RandomForestClassifier(
            n_estimators=100, max_depth=8, min_samples_leaf=5,
            class_weight={0: 1, 1: n_neg / n_pos},
            n_jobs=-1, random_state=42)
        clf.fit(X_train, y_train, sample_weight=weights)
        return clf

    elif name == 'Extra Trees':
        n_pos = max(1, (y_train == 1).sum())
        n_neg = max(1, (y_train == 0).sum())
        clf = ExtraTreesClassifier(
            n_estimators=100, max_depth=8, min_samples_leaf=5,
            class_weight={0: 1, 1: n_neg / n_pos},
            n_jobs=-1, random_state=42)
        clf.fit(X_train, y_train, sample_weight=weights)
        return clf

    elif name == 'Gradient Boost':
        clf = GradientBoostingClassifier(
            n_estimators=100, max_depth=5, learning_rate=0.05,
            subsample=0.8, random_state=42)
        clf.fit(X_train, y_train, sample_weight=weights)
        return clf

    elif name == 'Logistic Reg':
        n_pos = max(1, (y_train == 1).sum())
        n_neg = max(1, (y_train == 0).sum())
        clf = LogisticRegression(
            C=1.0, max_iter=500,
            class_weight={0: 1, 1: n_neg / n_pos},
            random_state=42, n_jobs=-1)
        clf.fit(X_train, y_train, sample_weight=weights)
        return clf


def predict_proba(name: str, model, X_test: pd.DataFrame) -> np.ndarray:
    if name == 'LightGBM':
        return model.predict(X_test)
    else:
        return model.predict_proba(X_test)[:, 1]


# ─── Single-model simulation ──────────────────────────────────────────────────
def run_one_model(model_name: str, df: pd.DataFrame, le: LabelEncoder) -> list:
    """Run 12-month simulation for one model. Returns list of weekly p_auc values."""
    primary_path = f'_primary_{model_name.replace(" ", "_")}.bin'

    # Bootstrap on first WINDOW_SIZE rows
    boot_df = df.iloc[:WINDOW_SIZE].copy()
    mask = boot_df['type'].isin(le.classes_)
    boot_df.loc[mask, 'type'] = le.transform(boot_df.loc[mask, 'type'])
    boot_df.loc[~mask, 'type'] = 0
    boot_df['type'] = boot_df['type'].astype(int)
    boot_df = engineer_features(boot_df)

    weights = time_decay_weights(len(boot_df), len(boot_df))
    primary = train_model(model_name, boot_df[FEATURES], boot_df['isFraud'],
                          weights, primary_path)

    # Also train IsolationForest for anomaly boosting
    iso = IsolationForest(n_estimators=100, contamination=0.05, random_state=42)
    iso.fit(boot_df[FEATURES])

    if model_name == 'LightGBM':
        primary.save_model(primary_path)

    p_aucs = []
    total_ticks = 52000 // CHUNK_SIZE

    for tick in range(1, total_ticks + 1):
        start_idx = (tick - 1) * CHUNK_SIZE
        end_idx   = start_idx + WINDOW_SIZE + CHUNK_SIZE
        if end_idx > len(df):
            break

        chunk = df.iloc[start_idx:end_idx].copy()
        mask2 = chunk['type'].isin(le.classes_)
        chunk.loc[mask2, 'type'] = le.transform(chunk.loc[mask2, 'type'])
        chunk.loc[~mask2, 'type'] = 0
        chunk['type'] = chunk['type'].astype(int)
        chunk = engineer_features(chunk)

        test_df  = chunk.iloc[-CHUNK_SIZE:]
        train_df = chunk.iloc[:-CHUNK_SIZE]

        if len(test_df['isFraud'].unique()) < 2:
            p_aucs.append(p_aucs[-1] if p_aucs else 0.95)
            continue

        try:
            # Hybrid predict: boost score if IsolationForest flags anomaly
            lgb_preds = predict_proba(model_name, primary, test_df[FEATURES])
            iso_flags = iso.predict(test_df[FEATURES])
            hybrid    = np.where(iso_flags == -1, np.maximum(lgb_preds, 0.85), lgb_preds)
            p_auc = roc_auc_score(test_df['isFraud'], hybrid)
        except Exception:
            p_aucs.append(p_aucs[-1] if p_aucs else 0.95)
            continue

        p_aucs.append(p_auc)

        # Three-tier regime
        if p_auc >= HEALTHY_FLOOR:
            continue  # HEALTHY — do nothing

        w = time_decay_weights(len(train_df), len(train_df))
        use_init = (model_name == 'LightGBM') and (p_auc >= DRIFT_FLOOR)
        active_window = WINDOW_SIZE if p_auc >= DRIFT_FLOOR else WINDOW_SIZE // 4
        t_df = train_df.iloc[-active_window:] if active_window < len(train_df) else train_df
        ww   = time_decay_weights(len(t_df), len(t_df))

        try:
            candidate = train_model(model_name, t_df[FEATURES], t_df['isFraud'],
                                    ww, primary_path)
            c_preds   = predict_proba(model_name, candidate, test_df[FEATURES])
            c_hybrid  = np.where(iso_flags == -1, np.maximum(c_preds, 0.85), c_preds)
            c_auc     = roc_auc_score(test_df['isFraud'], c_hybrid)

            promote_threshold = 0.02 if p_auc >= DRIFT_FLOOR else 0.0
            if c_auc > p_auc + promote_threshold:
                primary = candidate
                iso.fit(t_df[FEATURES])
                if model_name == 'LightGBM':
                    primary.save_model(primary_path)
        except Exception:
            pass

    # Cleanup
    if os.path.exists(primary_path):
        os.remove(primary_path)

    return p_aucs


# ─── Plotting ─────────────────────────────────────────────────────────────────
def plot_shootout(all_results: dict, model_registry: dict):
    """
    all_results: {dataset_name: {model_name: [p_aucs]}}
    """
    fig = plt.figure(figsize=(22, 14))
    fig.patch.set_facecolor('#0D1117')
    gs = gridspec.GridSpec(2, 2, figure=fig, hspace=0.50, wspace=0.35)
    axs = [fig.add_subplot(gs[i // 2, i % 2]) for i in range(4)]

    for ax, ds in zip(axs, DATASETS):
        ds_name = ds['name']
        ax.set_facecolor('#161B22')
        ax.tick_params(colors='#8B949E', labelsize=8)
        for spine in ax.spines.values():
            spine.set_edgecolor('#30363D')

        model_results = all_results.get(ds_name, {})
        for model_name, cfg in model_registry.items():
            aucs = model_results.get(model_name, [])
            if not aucs:
                continue
            ticks  = list(range(1, len(aucs) + 1))
            smooth = pd.Series(aucs).rolling(3, min_periods=1, center=True).mean().tolist()
            ax.plot(ticks, aucs,   color=cfg['color'], linewidth=0.5, alpha=0.20, ls='-')
            ax.plot(ticks, smooth, color=cfg['color'], linewidth=cfg['lw'],
                    linestyle=cfg['ls'], label=model_name, alpha=0.95)

        ax.axhline(HEALTHY_FLOOR, color='#FFD700', linewidth=1.0, linestyle='--',
                   alpha=0.5, label='HEALTHY (0.95)')
        ax.axhline(DRIFT_FLOOR,   color='#FF6B6B', linewidth=1.0, linestyle=':',
                   alpha=0.5, label='DRIFT (0.90)')

        ax.set_ylim(0.30, 1.05)
        ax.set_xlim(1, 52)
        ax.set_title(ds_name, fontsize=13, fontweight='bold', color='#E6EDF3', pad=10)
        ax.set_xlabel('Week in Production', fontsize=9, color='#8B949E')
        ax.set_ylabel('ROC AUC', fontsize=9, color='#8B949E')
        ax.legend(fontsize=7, loc='lower right', ncol=2,
                  facecolor='#161B22', labelcolor='#E6EDF3', framealpha=0.8,
                  edgecolor='#30363D')

    fig.suptitle('Model Shootout: Adaptive Three-Tier Pipeline\nLightGBM vs XGBoost vs CatBoost vs RF vs ET vs GBM vs LR',
                 fontsize=15, fontweight='bold', color='#E6EDF3', y=0.98)

    plt.savefig(OUT_PNG, dpi=200, bbox_inches='tight', facecolor=fig.get_facecolor())
    print(f"\nShootout graph saved -> {OUT_PNG}")


# ─── Main ─────────────────────────────────────────────────────────────────────
if __name__ == '__main__':
    model_registry = get_model_registry()
    print(f"Models in shootout: {list(model_registry.keys())}\n")

    all_results = {}

    for ds in DATASETS:
        ds_name = ds['name']
        path    = ds['path']
        print(f"\n{'='*65}")
        print(f"  DATASET: {ds_name}")
        print(f"{'='*65}")

        if not os.path.exists(path):
            print(f"  [SKIP] Not found: {path}")
            all_results[ds_name] = {}
            continue

        df = pd.read_csv(path).sort_values('step').reset_index(drop=True)
        print(f"  Rows: {len(df):,}  |  Fraud rate: {df['isFraud'].mean()*100:.2f}%")

        # Fit a shared label encoder per dataset
        le = LabelEncoder()
        le.fit(df['type'].astype(str))

        all_results[ds_name] = {}

        for model_name in model_registry:
            print(f"\n  -- {model_name} --")
            try:
                aucs = run_one_model(model_name, df.copy(), le)
                all_results[ds_name][model_name] = aucs
                valid = [a for a in aucs if a is not None]
                print(f"     Mean AUC: {np.mean(valid):.4f}  |  Min: {min(valid):.4f}  |  Weeks: {len(aucs)}")
            except Exception as e:
                print(f"     [ERROR] {e}")
                all_results[ds_name][model_name] = []

    plot_shootout(all_results, model_registry)
    print("\nShootout complete.")
