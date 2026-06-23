from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
DATA_DIR = ROOT / "data"
ARTIFACTS_DIR = ROOT / "artifacts"

RANDOM_STATE = 42
TEST_SIZE = 0.2
CV_FOLDS = 5

# Tree structure shared by weak baselines (class weights applied separately).
BASELINE_TREE = {
    "n_estimators": 5,
    "max_depth": 3,
    "min_samples_split": 80,
    "min_samples_leaf": 40,
    "max_features": 1,
}

# Skewed weights stop the model from defaulting to the 76% majority class.
BASELINE_RF = {
    **BASELINE_TREE,
    "class_weight": {0: 1, 1: 5},
}

BASELINE_LGBM = {
    "n_estimators": 10,
    "max_depth": 3,
    "num_leaves": 6,
    "scale_pos_weight": 9,
}
