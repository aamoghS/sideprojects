# Random Forest & LightGBM optimization lab

Train deliberately weak tree models (~60% accuracy), then improve them with hyperparameter tuning and different RF variants.

All Python — no API layer, no Docker.

## Dataset

[UCI Adult](https://archive.ics.uci.edu/ml/datasets/adult) income prediction (>50K vs <=50K). Downloaded on first run.

## Setup

```powershell
cd optimization
python -m venv .venv
.\.venv\Scripts\Activate.ps1
pip install -r requirements.txt
```

## Run

```powershell
# Full pipeline: baselines → tuning → comparison chart
python -m src.main all

# Fast smoke run (fewer search iterations)
python -m src.main all --quick

# Individual steps
python -m src.main baseline
python -m src.main tune
python -m src.main compare
```

## What it does

### Phase 1 — Baselines (~60% target)

Weak setup on purpose: only 3 numeric features (age, hours-per-week, education-num), shallow trees, and skewed class weights so accuracy lands near 60% instead of defaulting to the 76% majority class.

### Phase 2 — Tuning

Run `python -m src.main methods` to see the full catalog. Ten methods are wired up:

| ID | Method | Model | What it tunes |
|----|--------|-------|---------------|
| `rf_randomized` | Randomized Search | Random Forest | n_estimators, depth, min_samples, max_features, bootstrap, class_weight |
| `rf_grid` | Grid Search | Random Forest | Focused exhaustive grid over key RF params |
| `rf_halving` | Successive Halving | Random Forest | Same space as random search, but kills weak configs early |
| `extra_trees_randomized` | Randomized Search | Extra Trees | Same as RF — random split points instead of optimal |
| `rf_optuna_variants` | Optuna TPE + picker | RF family | Picks best variant (standard / balanced / bootstrap-off) + params |
| `lgbm_randomized` | Randomized Search | LightGBM | lr, leaves, subsample, regularization, scale_pos_weight |
| `lgbm_optuna` | Optuna TPE | LightGBM | Bayesian-style search; learns from past trials |
| `lgbm_optuna_pruned` | Optuna + early stopping | LightGBM | Stops boosting when validation loss plateaus |
| `lgbm_f1_optuna` | Optuna (F1 objective) | LightGBM | Optimizes F1 + scale_pos_weight for imbalanced labels |
| `rf_threshold` | Threshold sweep | Random Forest | Finds best probability cutoff after training (no retrain) |

```powershell
python -m src.main methods              # print full catalog with explanations
python -m src.main tune                 # default 4 methods (fast enough for daily use)
python -m src.main tune --all-methods   # run all 10
python -m src.main tune --method lgbm_f1_optuna   # run one
```

**Other tuning ideas** (not wired — add in `src/tune.py` if you need them):

- Feature selection (RFE / SelectFromModel) inside the search loop
- SMOTE or other resampling as a pipeline step
- Stacking / voting ensembles of RF + LGBM
- Full Hyperband via Optuna's `HyperbandPruner`
- Auto-sklearn or FLAML for fully automated model + preprocess search

### Output

Results and models land in `artifacts/`:

- `baseline_results.json` / `tuned_results.json` — metrics
- `best_params.json` — winning hyperparameters
- `accuracy_comparison.png` — baseline vs tuned bar chart
- `*.joblib` — saved pipelines

## Layout

```
optimization/
  src/
    config.py      Shared constants and weak baseline params
    data.py        Adult dataset load + preprocessing
    models.py      RF variants + LightGBM factories
    baseline.py    Train weak models
    tune.py        RandomizedSearch + Optuna
    compare.py     Side-by-side metrics + chart
    main.py        CLI
  artifacts/       Generated on run (gitignored)
  requirements.txt
```

## RF variants explained

- **random_forest** — standard bagged decision trees
- **extra_trees** — random splits instead of optimal splits (often generalizes better)
- **balanced_rf** — adjusts for class imbalance (>50K is ~24% of labels)
- **bootstrap_false_rf** — each tree sees the full dataset (no bootstrap sampling)

## Extending

- Swap the dataset in `src/data.py` — keep the same pipeline shape
- Add search ranges in `src/tune.py`
- Register new RF factories in `src/models.py` under `RF_VARIANTS`
