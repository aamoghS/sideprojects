# research

ML experiments around concept drift and adaptive retraining on synthetic fraud datasets.

The core idea: train a primary LightGBM model on an initial window of transactions, then stream new data in chunks. An `AdaptiveRetrainer` watches for drift, trains candidate models on a sliding window, and shadow-deploys them before promoting a winner. Synthetic datasets (`dataset_ecommerce.csv`, `dataset_crypto.csv`, etc.) simulate different fraud patterns and drift profiles so you can compare how the same pipeline behaves across domains.

## Scripts

- `train.py` — bootstrap the primary model on PaySim banking data
- `retrain_pipeline.py` — incremental retraining with drift detection
- `run_benchmark.py` — compare adaptive retraining across four dataset types
- `run_model_shootout.py` — pit RF, LightGBM, logistic regression, isolation forest, etc. through the same pipeline
- `run_forecast_shootout.py` / `run_*_simulation.py` — time-series forecast comparisons
- `generate_datasets.py` / `generate_paysim.py` — create synthetic CSVs

## Setup

```bash
cd research
python -m venv .venv
.\.venv\Scripts\Activate.ps1   # or source .venv/bin/activate
pip install -r requirements.txt
```

Generate datasets first if the CSVs are missing:

```bash
python generate_paysim.py
python generate_datasets.py
```

Then run whatever script you need, e.g. `python run_benchmark.py`.
