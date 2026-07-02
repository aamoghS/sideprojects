import os
import sys
import logging
import requests
import joblib
import pandas as pd
import numpy as np
import lightgbm as lgb
from sklearn.metrics import roc_auc_score
from sklearn.ensemble import IsolationForest

logging.basicConfig(level=logging.INFO, format='%(asctime)s [%(levelname)s] %(message)s')
logger = logging.getLogger(__name__)

class AdaptiveRetrainer:
    """
    Handles incremental model updates and drift detection via adaptive windowing.
    Implements ensemble decay and shadow deployment gating.
    """
    def __init__(self, data_path: str, primary_path: str, candidate_path: str, encoder_path: str):
        self.data_path = data_path
        self.primary_path = primary_path
        self.candidate_path = candidate_path
        self.primary_if_path = "primary_if.joblib"
        self.candidate_if_path = "candidate_if.joblib"
        self.encoder_path = encoder_path
        self.features = ['step', 'type', 'amount', 'oldbalanceOrg', 'newbalanceOrig', 'oldbalanceDest', 'newbalanceDest', 'balance_diff', 'amount_to_oldbalance_ratio', 'dest_balance_diff']

    def engineer_features(self, df: pd.DataFrame) -> pd.DataFrame:
        df['balance_diff'] = df['newbalanceOrig'] - df['oldbalanceOrg']
        df['amount_to_oldbalance_ratio'] = df['amount'] / (df['oldbalanceOrg'] + 1)
        df['dest_balance_diff'] = df['newbalanceDest'] - df['oldbalanceDest']
        return df

    def _apply_time_decay(self, df: pd.DataFrame, window_size: int) -> pd.Series:
        decay_rate = np.log(2) / (window_size / 2)
        relative_age = np.arange(len(df))[::-1]
        weights = np.exp(-decay_rate * relative_age)
        return weights

    def _train_candidate(self, train_df: pd.DataFrame, weights: pd.Series, use_init_model: bool = True) -> tuple[lgb.Booster, IsolationForest]:
        train_data = lgb.Dataset(train_df[self.features], label=train_df['isFraud'], weight=weights)
        params = {
            'objective': 'binary',
            'metric': 'auc',
            'learning_rate': 0.05,
            'num_leaves': 15,
            'max_depth': 4,
            'verbose': -1
        }
        
        lgbm_model = lgb.train(
            params,
            train_data,
            num_boost_round=50,
            init_model=self.primary_path if use_init_model else None
        )
        
        iso_model = IsolationForest(n_estimators=100, contamination=0.05, random_state=42)
        iso_model.fit(train_df[self.features])
        
        return lgbm_model, iso_model

    def _hybrid_predict(self, lgbm: lgb.Booster, iso: IsolationForest, test_df: pd.DataFrame) -> np.ndarray:
        X_test = test_df[self.features]
        lgb_preds = lgbm.predict(X_test)
        iso_preds = iso.predict(X_test)
        hybrid_preds = np.where(iso_preds == -1, np.maximum(lgb_preds, 0.85), lgb_preds)
        return hybrid_preds

    def _evaluate(self, p_lgbm: lgb.Booster, p_iso: IsolationForest, c_lgbm: lgb.Booster, c_iso: IsolationForest, test_df: pd.DataFrame) -> tuple[float, float]:
        y_true = test_df['isFraud']
        p_pred = self._hybrid_predict(p_lgbm, p_iso, test_df)
        c_pred = self._hybrid_predict(c_lgbm, c_iso, test_df)
        
        p_auc = roc_auc_score(y_true, p_pred)
        c_auc = roc_auc_score(y_true, c_pred)
        
        return p_auc, c_auc

    def execute_pipeline(self, initial_window: int = 16000, test_size: int = 1000):
        try:
            df = pd.read_csv(self.data_path)
            encoder = joblib.load(self.encoder_path)
            
            mask = df['type'].isin(encoder.classes_)
            df.loc[mask, 'type'] = encoder.transform(df.loc[mask, 'type'])
            df.loc[~mask, 'type'] = 0
            df['type'] = df['type'].astype(int)
            
            df = self.engineer_features(df)

            if not os.path.exists(self.primary_path) or not os.path.exists(self.primary_if_path):
                raise FileNotFoundError(f"Primary models not found.")
                
            primary_lgbm = lgb.Booster(model_file=self.primary_path)
            primary_iso = joblib.load(self.primary_if_path)
            
            test_df = df.iloc[-test_size:].copy()
            if len(test_df['isFraud'].unique()) < 2:
                return None, None, False

            drift_preds = self._hybrid_predict(primary_lgbm, primary_iso, test_df)
            p_auc = roc_auc_score(test_df['isFraud'], drift_preds)
            
            # Quant-grade three-tier regime:
            # HEALTHY  (>= 0.95): Do nothing, model is strong.
            # WATCH    (0.90-0.95): Early warning — train a shadow candidate silently.
            #                       Only promote if candidate beats primary by > 0.02 margin.
            # DRIFT    (< 0.90): Act immediately — hot-swap to best available model right now.
            
            HEALTHY_FLOOR = 0.95
            DRIFT_FLOOR   = 0.90

            if p_auc >= HEALTHY_FLOOR:
                return p_auc, None, False  # HEALTHY: hold position

            # Build training window (shrink aggressively in full drift for recency bias)
            use_init_model = p_auc >= DRIFT_FLOOR
            active_window = initial_window if use_init_model else initial_window // 4
            if active_window < len(df) - test_size:
                train_df = df.iloc[-(active_window + test_size):-test_size].copy()
            else:
                train_df = df.iloc[:-test_size].copy()

            weights = self._apply_time_decay(train_df, len(train_df))
            candidate_lgbm, candidate_iso = self._train_candidate(train_df, weights, use_init_model)
            _, c_auc = self._evaluate(primary_lgbm, primary_iso, candidate_lgbm, candidate_iso, test_df)

            if p_auc >= DRIFT_FLOOR:
                # WATCH regime: only promote if candidate is meaningfully better (>2% margin)
                if c_auc > p_auc + 0.02:
                    candidate_lgbm.save_model(self.primary_path)
                    joblib.dump(candidate_iso, self.primary_if_path)
                    try:
                        requests.post("http://127.0.0.1:8000/reload_model", timeout=2)
                    except requests.exceptions.RequestException:
                        pass
                    return p_auc, c_auc, True
                return p_auc, c_auc, False
            else:
                # DRIFT regime: hot-swap immediately if candidate is any better at all
                if c_auc > p_auc:
                    candidate_lgbm.save_model(self.primary_path)
                    joblib.dump(candidate_iso, self.primary_if_path)
                    try:
                        requests.post("http://127.0.0.1:8000/reload_model", timeout=2)
                    except requests.exceptions.RequestException:
                        pass
                    return p_auc, c_auc, True
                return p_auc, c_auc, False

        except Exception as e:
            logger.error(f"Pipeline failure: {str(e)}", exc_info=True)
            return None, None, False
