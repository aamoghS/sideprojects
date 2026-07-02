import pandas as pd
from sklearn.preprocessing import LabelEncoder
import joblib
import lightgbm as lgb
import os
import logging

logging.basicConfig(level=logging.INFO, format='%(asctime)s [%(levelname)s] %(message)s')
logger = logging.getLogger(__name__)

def engineer_features(df: pd.DataFrame) -> pd.DataFrame:
    df['balance_diff'] = df['newbalanceOrig'] - df['oldbalanceOrg']
    df['amount_to_oldbalance_ratio'] = df['amount'] / (df['oldbalanceOrg'] + 1)
    df['dest_balance_diff'] = df['newbalanceDest'] - df['oldbalanceDest']
    return df

def train_initial_model(data_path="paysim.csv", model_path="primary_model.txt", encoder_path="label_encoder.joblib", nrows=16000):
    """
    Bootstraps the initial Primary model on a historical baseline of data.
    """
    logger.info(f"Bootstrapping Initial Primary Model on first {nrows} rows...")
    
    if not os.path.exists(data_path):
        logger.error(f"Dataset {data_path} not found.")
        return
        
    df = pd.read_csv(data_path).sort_values(by='step').reset_index(drop=True)
    
    # Train and save the LabelEncoder
    le = LabelEncoder()
    df['type'] = le.fit_transform(df['type'])
    joblib.dump(le, encoder_path)
    
    # Use the initial baseline window
    train_df = df.iloc[:nrows].copy()
    
    # Engineer Features
    train_df = engineer_features(train_df)
    
    features = ['step', 'type', 'amount', 'oldbalanceOrg', 'newbalanceOrig', 'oldbalanceDest', 'newbalanceDest', 'balance_diff', 'amount_to_oldbalance_ratio', 'dest_balance_diff']
    train_data = lgb.Dataset(train_df[features], label=train_df['isFraud'])
    
    params = {
        'objective': 'binary', 
        'metric': 'auc', 
        'learning_rate': 0.05, 
        'num_leaves': 15, 
        'max_depth': 4, 
        'verbose': -1
    }
    
    # Train the initial baseline LGBM
    model = lgb.train(params, train_data, num_boost_round=100)
    model.save_model(model_path)
    
    # Train the initial Isolation Forest
    logger.info("Training Unsupervised Isolation Forest...")
    from sklearn.ensemble import IsolationForest
    iso_forest = IsolationForest(n_estimators=100, contamination=0.05, random_state=42)
    iso_forest.fit(train_df[features])
    joblib.dump(iso_forest, "primary_if.joblib")
    
    logger.info(f"Baseline Primary Models successfully saved.")

if __name__ == "__main__":
    train_initial_model()
