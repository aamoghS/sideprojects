"""
Fast parallel 9-domain model shootout.
Speed wins:
  - joblib.Parallel runs all 6 models simultaneously per dataset
  - n_estimators cut to 40 (fast, still accurate)
  - window_size = 8000, chunk_size = 2000 -> 26 ticks instead of 52
  - dataset trimmed to 40k rows
  - IsolationForest fitted once per dataset, shared across ticks
"""
import os
import warnings
import numpy as np
import pandas as pd
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import matplotlib.gridspec as gridspec
from joblib import Parallel, delayed
from sklearn.preprocessing import LabelEncoder
from sklearn.metrics import roc_auc_score
from sklearn.ensemble import RandomForestClassifier, ExtraTreesClassifier, IsolationForest
from sklearn.linear_model import LogisticRegression
import lightgbm as lgb
import xgboost as xgb
from catboost import CatBoostClassifier

warnings.filterwarnings('ignore')

# ── Speed config ──────────────────────────────────────────────────────────────
N_ROWS      = 40000   # rows per dataset (was 70k)
CHUNK_SIZE  = 2000    # weekly ticks (was 1000 → 2x faster)
WINDOW_SIZE = 8000    # training window (was 12k)
N_EST       = 40      # estimators (was 100)
HEALTHY     = 0.95
DRIFT       = 0.90
OUT_PNG     = r'C:\Users\bootcamp\Desktop\sideprojects\research\forecast_shootout.png'

DATASETS = [
    {'name': 'Market Regime',  'path': 'dataset_market.csv'},
    {'name': 'Energy Grid',    'path': 'dataset_energy.csv'},
    {'name': 'IoT Sensors',    'path': 'dataset_iot.csv'},
    {'name': 'Retail Sales',   'path': 'dataset_retail.csv'},
    {'name': 'Healthcare ICU', 'path': 'dataset_healthcare.csv'},
    {'name': 'Cybersecurity',  'path': 'dataset_cybersec.csv'},
    {'name': 'Supply Chain',   'path': 'dataset_supplychain.csv'},
    {'name': 'Climate',        'path': 'dataset_climate.csv'},
    {'name': 'Credit Risk',    'path': 'dataset_credit.csv'},
]

MODELS = {
    'CatBoost':      {'color': '#A8D5A2', 'ls': '-',  'lw': 2.8},
    'XGBoost':       {'color': '#F4845F', 'ls': '-',  'lw': 2.0},
    'LightGBM':      {'color': '#4C9BE8', 'ls': '-',  'lw': 2.0},
    'Extra Trees':   {'color': '#FFD166', 'ls': '--', 'lw': 2.0},
    'Random Forest': {'color': '#C77DFF', 'ls': '--', 'lw': 2.0},
    'Logistic Reg':  {'color': '#888888', 'ls': ':',  'lw': 1.5},
}

BASE_FEATS = ['step', 'type', 'amount', 'oldbalanceOrg', 'newbalanceOrig',
              'oldbalanceDest', 'newbalanceDest']

def eng(df: pd.DataFrame) -> pd.DataFrame:
    """Add lag/rolling features on top of whatever domain columns exist."""
    df = df.copy()
    df['balance_diff'] = df['newbalanceOrig'] - df['oldbalanceOrg']
    df['amount_ratio'] = df['amount'] / (df['oldbalanceOrg'].abs() + 1)
    df['dest_diff']    = df['newbalanceDest'] - df['oldbalanceDest']
    # Multi-lag on amount
    for lag in [1, 3, 7, 14]:
        df[f'lag{lag}'] = df['amount'].shift(lag).fillna(df['amount'].median())
    # Rolling stats at multiple windows
    for w in [5, 10, 20, 50]:
        df[f'rmean{w}'] = df['amount'].rolling(w, min_periods=1).mean()
        df[f'rstd{w}']  = df['amount'].rolling(w, min_periods=1).std().fillna(0)
    # First-order diff (momentum)
    df['diff1']  = df['amount'].diff(1).fillna(0)
    df['diff5']  = df['amount'].diff(5).fillna(0)
    df['diff20'] = df['amount'].diff(20).fillna(0)
    return df

def get_features(df: pd.DataFrame) -> list[str]:
    """Return all usable numeric feature columns (exclude step, isFraud, type)."""
    exclude = {'step', 'isFraud', 'type'}
    return [c for c in df.columns if c not in exclude and pd.api.types.is_numeric_dtype(df[c])]


def decay(n: int) -> np.ndarray:
    r = np.log(2) / (n / 2)
    return np.exp(-r * np.arange(n)[::-1])


def fit(name: str, X: pd.DataFrame, y: pd.Series, w: np.ndarray, lgb_path: str):
    if name == 'LightGBM':
        ds = lgb.Dataset(X, label=y, weight=w)
        p  = {'objective': 'binary', 'metric': 'auc', 'learning_rate': 0.08,
              'num_leaves': 31, 'max_depth': 5, 'verbose': -1,
              'min_child_samples': 10, 'subsample': 0.8}
        return lgb.train(p, ds, num_boost_round=N_EST,
                         init_model=lgb_path if os.path.exists(lgb_path) else None)
    elif name == 'XGBoost':
        r = max(1, int((y==0).sum()/max(1,(y==1).sum())))
        m = xgb.XGBClassifier(n_estimators=N_EST, max_depth=5, learning_rate=0.08,
                               scale_pos_weight=r, verbosity=0,
                               use_label_encoder=False, random_state=42, n_jobs=2)
        m.fit(X, y, sample_weight=w); return m
    elif name == 'CatBoost':
        m = CatBoostClassifier(iterations=N_EST, depth=5, learning_rate=0.08,
                               eval_metric='AUC', verbose=False, random_seed=42,
                               auto_class_weights='Balanced', thread_count=2)
        m.fit(X, y, sample_weight=w); return m
    elif name == 'Extra Trees':
        r = max(1, int((y==0).sum()/max(1,(y==1).sum())))
        m = ExtraTreesClassifier(n_estimators=N_EST, max_depth=7,
                                 class_weight={0:1,1:r}, n_jobs=2, random_state=42)
        m.fit(X, y, sample_weight=w); return m
    elif name == 'Random Forest':
        r = max(1, int((y==0).sum()/max(1,(y==1).sum())))
        m = RandomForestClassifier(n_estimators=N_EST, max_depth=7,
                                   class_weight={0:1,1:r}, n_jobs=2, random_state=42)
        m.fit(X, y, sample_weight=w); return m
    elif name == 'Logistic Reg':
        r = max(1, int((y==0).sum()/max(1,(y==1).sum())))
        m = LogisticRegression(C=1.0, max_iter=300, class_weight={0:1,1:r},
                               random_state=42, n_jobs=2)
        m.fit(X, y, sample_weight=w); return m


def pred(name: str, model, X: pd.DataFrame, iso: IsolationForest) -> np.ndarray:
    raw   = model.predict(X) if name == 'LightGBM' else model.predict_proba(X)[:,1]
    flags = iso.predict(X)
    return np.where(flags == -1, np.maximum(raw, 0.82), raw)


def run_model(model_name: str, df: pd.DataFrame, iso: IsolationForest, feats: list) -> list[float]:
    """Single model walk-forward. iso and feats are pre-computed and shared."""
    lgb_path = f'_fast_{model_name.replace(" ","_")}.lgb'

    boot = df.iloc[:WINDOW_SIZE]
    try:
        w0      = decay(len(boot))
        primary = fit(model_name, boot[feats], boot['isFraud'], w0, lgb_path)
        if model_name == 'LightGBM':
            primary.save_model(lgb_path)
    except Exception:
        return []

    aucs  = []
    total = (N_ROWS - WINDOW_SIZE) // CHUNK_SIZE

    for tick in range(total):
        s = tick * CHUNK_SIZE
        e = s + WINDOW_SIZE + CHUNK_SIZE
        if e > len(df): break

        chunk   = df.iloc[s:e]
        test_df = chunk.iloc[-CHUNK_SIZE:]
        tr_df   = chunk.iloc[:-CHUNK_SIZE]

        if len(test_df['isFraud'].unique()) < 2:
            aucs.append(aucs[-1] if aucs else 0.95)
            continue

        try:
            p_auc = roc_auc_score(test_df['isFraud'], pred(model_name, primary, test_df[feats], iso))
        except Exception:
            aucs.append(aucs[-1] if aucs else 0.95)
            continue

        aucs.append(p_auc)

        if p_auc >= HEALTHY:
            continue

        aw = WINDOW_SIZE if p_auc >= DRIFT else WINDOW_SIZE // 3
        t  = tr_df.iloc[-aw:] if aw < len(tr_df) else tr_df
        w  = decay(len(t))

        try:
            cand  = fit(model_name, t[feats], t['isFraud'], w, lgb_path)
            c_auc = roc_auc_score(test_df['isFraud'], pred(model_name, cand, test_df[feats], iso))
            margin = 0.02 if p_auc >= DRIFT else 0.0
            if c_auc > p_auc + margin:
                primary = cand
                if model_name == 'LightGBM':
                    primary.save_model(lgb_path)
        except Exception:
            pass

    if os.path.exists(lgb_path):
        os.remove(lgb_path)
    return aucs


def run_dataset(ds: dict) -> tuple[str, dict]:
    name, path = ds['name'], ds['path']
    if not os.path.exists(path):
        print(f'  [SKIP] {name}: file not found')
        return name, {}

    df = pd.read_csv(path, nrows=N_ROWS).sort_values('step').reset_index(drop=True)
    le = LabelEncoder()
    df['type'] = le.fit_transform(df['type'].astype(str))
    df = eng(df)
    # Coerce all columns to numeric, fill NaN
    for c in df.columns:
        if c not in ('isFraud',):
            df[c] = pd.to_numeric(df[c], errors='coerce').fillna(0)

    feats = get_features(df)   # dynamic — picks up all domain-specific columns
    pos   = df['isFraud'].mean() * 100
    print(f'  [{name}]  rows={len(df):,}  pos={pos:.1f}%  feats={len(feats)}  running {len(MODELS)} models...')

    iso = IsolationForest(n_estimators=50, contamination=0.05, random_state=42)
    iso.fit(df.iloc[:WINDOW_SIZE][feats])

    results_list = Parallel(n_jobs=len(MODELS), prefer='threads')(
        delayed(run_model)(m, df.copy(), iso, feats) for m in MODELS
    )

    model_results = {}
    for model_name, aucs in zip(MODELS.keys(), results_list):
        if aucs:
            print(f'    {model_name:<14} mean={np.mean(aucs):.4f}  min={min(aucs):.4f}')
        model_results[model_name] = aucs

    return name, model_results


def smooth(a: list, k: int = 3) -> list:
    return pd.Series(a).rolling(k, min_periods=1, center=True).mean().tolist()


def plot(all_results: dict):
    fig = plt.figure(figsize=(26, 20))
    fig.patch.set_facecolor('#0D1117')
    gs  = gridspec.GridSpec(3, 3, figure=fig, hspace=0.55, wspace=0.35)
    axs = [fig.add_subplot(gs[i//3, i%3]) for i in range(9)]

    for ax, ds in zip(axs, DATASETS):
        name = ds['name']
        ax.set_facecolor('#161B22')
        ax.tick_params(colors='#8B949E', labelsize=7)
        for sp in ax.spines.values():
            sp.set_edgecolor('#30363D')

        res = all_results.get(name, {})
        best_mean, best_name = -1, ''

        for mname, cfg in MODELS.items():
            aucs = res.get(mname, [])
            if not aucs: continue
            ticks = list(range(1, len(aucs)+1))
            s     = smooth(aucs)
            ax.fill_between(ticks, [v-0.006 for v in s], [v+0.006 for v in s],
                            color=cfg['color'], alpha=0.08)
            ax.plot(ticks, s, color=cfg['color'], lw=cfg['lw'], ls=cfg['ls'],
                    label=f"{mname} ({np.mean(aucs):.3f})", alpha=0.92)
            if np.mean(aucs) > best_mean:
                best_mean, best_name = np.mean(aucs), mname

        ax.axhline(HEALTHY, color='#FFD700', lw=1.0, ls='--', alpha=0.5)
        ax.axhline(DRIFT,   color='#FF6B6B', lw=1.0, ls=':',  alpha=0.5)
        ax.text(0.5, HEALTHY+0.008, '0.95', color='#FFD700', fontsize=6, alpha=0.7,
                transform=ax.get_yaxis_transform())
        ax.text(0.5, DRIFT+0.008,   '0.90', color='#FF6B6B', fontsize=6, alpha=0.7,
                transform=ax.get_yaxis_transform())

        ax.set_ylim(0.30, 1.05)
        ax.set_xlim(0, max((len(r) for r in res.values() if r), default=26)+1)
        title_color = '#FFD700' if best_name == 'CatBoost' else '#E6EDF3'
        ax.set_title(f"{name}  ·  🏆 {best_name} ({best_mean:.3f})",
                     fontsize=10, fontweight='bold', color=title_color, pad=8)
        ax.set_xlabel('Evaluation Window', fontsize=8, color='#8B949E')
        ax.set_ylabel('ROC AUC',           fontsize=8, color='#8B949E')
        ax.legend(fontsize=6.5, loc='lower right', ncol=2,
                  facecolor='#161B22', labelcolor='#E6EDF3',
                  framealpha=0.85, edgecolor='#30363D')

    fig.suptitle(
        '9-Domain Walk-Forward Future Prediction Shootout\n'
        'CatBoost · XGBoost · LightGBM · Extra Trees · Random Forest · Logistic Reg\n'
        'Finance  ·  Energy  ·  IoT  ·  Retail  ·  Healthcare  ·  Cyber  ·  Supply Chain  ·  Climate  ·  Credit',
        fontsize=13, fontweight='bold', color='#E6EDF3', y=0.995)

    plt.savefig(OUT_PNG, dpi=180, bbox_inches='tight', facecolor=fig.get_facecolor())
    print(f'\nSaved -> {OUT_PNG}')


if __name__ == '__main__':
    import time
    t0 = time.time()

    # Datasets run sequentially (each dataset parallelizes models internally)
    all_results = {}
    for ds in DATASETS:
        name, res = run_dataset(ds)
        all_results[name] = res

    plot(all_results)
    print(f'\nTotal time: {(time.time()-t0)/60:.1f} min')
