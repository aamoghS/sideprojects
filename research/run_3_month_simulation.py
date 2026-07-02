import pandas as pd
import os
import lightgbm as lgb
from retrain_pipeline import AdaptiveRetrainer
import logging
from train import train_initial_model

logging.getLogger('retrain_pipeline').setLevel(logging.CRITICAL)

def inject_drift(df: pd.DataFrame):
    """Simulates 3-month concept drift (Wave 1 and Wave 2)."""
    # Wave 1: Small Payments
    mask1 = (df.index >= 32000) & (df.index < 40000) & (df['type'] == 'PAYMENT') & (df['amount'] < 20000)
    df.loc[df[mask1].sample(frac=0.15, random_state=42).index, 'isFraud'] = 1

    # Wave 2: Large Debits
    mask2 = (df.index >= 48000) & (df.index < 56000) & (df['type'] == 'DEBIT') & (df['amount'] > 1000)
    df.loc[df[mask2].sample(frac=0.8, random_state=42).index, 'isFraud'] = 1
    return df

def execute_simulation():
    master_df = pd.read_csv("paysim.csv").sort_values(by='step').reset_index(drop=True)
    
    # Initialize the baseline model (simulating pre-production setup)
    train_initial_model(nrows=16000)
    
    master_df = inject_drift(master_df)
    
    chunk_size = 4000
    window_size = 16000
    
    retrainer = AdaptiveRetrainer(
        data_path="temp_3m_data.csv",
        primary_path="primary_model.txt",
        candidate_path="candidate_model.txt",
        encoder_path="label_encoder.joblib"
    )
    
    promotions = 0
    rejections = 0
    
    print(f"\n--- 3-Month Production Pipeline Simulation ---")
    print(f"{'Week':<5} | {'Result':<20}")
    print("-" * 30)
    
    for week in range(1, 13):
        start_idx = (week - 1) * chunk_size
        end_idx = start_idx + window_size + chunk_size
        if end_idx > len(master_df):
            break
            
        weekly_df = master_df.iloc[start_idx:end_idx].copy()
        weekly_df.to_csv("temp_3m_data.csv", index=False)
        
        mtime_before = os.path.getmtime("primary_model.txt")
        retrainer.execute_pipeline(initial_window=window_size, test_size=chunk_size)
        mtime_after = os.path.getmtime("primary_model.txt")
        
        if mtime_after > mtime_before:
            print(f"Wk {week:02d} | WIN (Promoted)")
            promotions += 1
        else:
            print(f"Wk {week:02d} | LOSE (Retained)")
            rejections += 1
            
        if os.path.exists("temp_3m_data.csv"):
            os.remove("temp_3m_data.csv")
            
    print(f"\nTotal Promotions: {promotions}")
    print(f"Total Degradations Prevented: {rejections}")

if __name__ == "__main__":
    execute_simulation()
