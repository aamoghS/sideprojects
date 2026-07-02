import pandas as pd
import numpy as np
from sklearn.preprocessing import LabelEncoder
import os
import joblib
import lightgbm as lgb
from retrain_pipeline import AdaptiveRetrainer
import logging

logging.getLogger('retrain_pipeline').setLevel(logging.CRITICAL)

def initialize_baseline(df: pd.DataFrame):
    """Bootstraps the initial primary model on Q1 data."""
    le = LabelEncoder()
    df['type'] = le.fit_transform(df['type'])
    joblib.dump(le, "label_encoder.joblib")
    
    train_df = df.iloc[:12000].copy()
    features = ['step', 'type', 'amount', 'oldbalanceOrg', 'newbalanceOrig', 'oldbalanceDest', 'newbalanceDest']
    train_data = lgb.Dataset(train_df[features], label=train_df['isFraud'])
    params = {'objective': 'binary', 'metric': 'auc', 'learning_rate': 0.05, 'num_leaves': 15, 'max_depth': 4, 'verbose': -1}
    
    model = lgb.train(params, train_data, num_boost_round=100)
    model.save_model("primary_model.txt")

def inject_drift(df: pd.DataFrame):
    """Simulates realistic concept drift (fraud rings mutating over time)."""
    mask1 = (df.index >= 16000) & (df.index < 24000) & (df['type'] == 'PAYMENT') & (df['amount'] < 20000)
    df.loc[df[mask1].sample(frac=0.2, random_state=42).index, 'isFraud'] = 1
    
    mask2 = (df.index >= 28000) & (df.index < 36000) & (df['type'] == 'DEBIT') & (df['amount'] > 1000)
    df.loc[df[mask2].sample(frac=0.8, random_state=42).index, 'isFraud'] = 1

    mask3 = (df.index >= 40000) & (df.index < 48000) & (df['type'] == 'TRANSFER') & (df['oldbalanceOrg'] == 0)
    df.loc[df[mask3].sample(frac=0.3, random_state=42).index, 'isFraud'] = 1

    mask4 = (df.index >= 52000) & (df.index < 60000) & (df['type'] == 'CASH_IN')
    df.loc[df[mask4].sample(frac=0.1, random_state=42).index, 'isFraud'] = 1
    return df

def execute_simulation():
    print("Reading paysim.csv...")
    master_df = pd.read_csv("paysim.csv").sort_values(by='step').reset_index(drop=True)
    
    # Bootstrap initial models using the updated train.py
    print("Bootstrapping models...")
    from train import train_initial_model
    train_initial_model(nrows=12000)
    
    print("Injecting drift...")
    master_df = inject_drift(master_df)
    
    chunk_size = 1000 # Weekly micro-batching
    window_size = 12000
    
    retrainer = AdaptiveRetrainer(
        data_path="temp_12m_data.csv",
        primary_path="primary_model.txt",
        candidate_path="candidate_model.txt",
        encoder_path="label_encoder.joblib"
    )
    
    promotions = 0
    rejections = 0
    
    print(f"{'Tick':<5} | {'Result':<20}")
    print("-" * 30)
    
    import matplotlib
    matplotlib.use('Agg')
    import seaborn as sns
    sns.set_theme(style="darkgrid")
    import matplotlib.pyplot as plt
    print("Dependencies loaded, starting loop...")
    
    ticks = []
    p_aucs = []
    c_aucs = []
    promotions_t = []
    
    total_ticks = 52000 // chunk_size
    
    for tick in range(1, total_ticks + 1):
        start_idx = (tick - 1) * chunk_size
        end_idx = start_idx + window_size + chunk_size
        if end_idx > len(master_df):
            break
            
        temp_df = master_df.iloc[start_idx:end_idx].copy()
        temp_df.to_csv("temp_12m_data.csv", index=False)
        
        p_auc, c_auc, promoted = retrainer.execute_pipeline(initial_window=window_size, test_size=chunk_size)
        
        ticks.append(tick)
        if p_auc is not None:
            p_aucs.append(p_auc)
            if c_auc is not None:
                c_aucs.append(c_auc)
                regime = "WATCH" if p_auc >= 0.90 else "DRIFT"
                if promoted:
                    promotions_t.append(tick)
                    print(f"Tk {tick:03d} | [{regime}] WIN (Promoted) P:{p_auc:.3f} -> C:{c_auc:.3f}")
                    promotions += 1
                else:
                    print(f"Tk {tick:03d} | [{regime}] HOLD (Retained) P:{p_auc:.3f} C:{c_auc:.3f}")
                    rejections += 1
            else:
                c_aucs.append(np.nan)
                print(f"Tk {tick:03d} | [HEALTHY] STATIC P:{p_auc:.3f}")
        else:
            p_aucs.append(p_aucs[-1] if p_aucs else 0.95)
            c_aucs.append(c_aucs[-1] if c_aucs else 0.95)
            print(f"Tk {tick:03d} | SKIP (Insufficient Variance)")
            
        if os.path.exists("temp_12m_data.csv"):
            os.remove("temp_12m_data.csv")
            
    # Plotting
    sns.set_theme(style="darkgrid")
    plt.figure(figsize=(14, 7))
    
    # Plot primary and candidate
    plt.plot(ticks, p_aucs, label='Production API (Primary)', color='#1f77b4', linewidth=2.5, zorder=3)
    plt.plot(ticks, c_aucs, label='Shadow Model (Candidate)', color='#ff7f0e', linestyle='--', linewidth=1.5, alpha=0.8, zorder=2)
    

    # Highlight Drift Waves
    plt.axvspan(16, 24, color='#d62728', alpha=0.15, label='Wave 1 (Small Payments)')
    plt.axvspan(28, 36, color='#9467bd', alpha=0.15, label='Wave 2 (Large Debits)')
    plt.axvspan(40, 48, color='#8c564b', alpha=0.15, label='Wave 3 (Empty Transfers)')
    plt.axvspan(52, 60, color='#e377c2', alpha=0.15, label='Wave 4 (Cash In Frauds)')
    
    plt.title('12-Month Fraud Detection System Performance under Concept Drift', fontsize=16, fontweight='bold', pad=20)
    plt.xlabel('Weeks in Production', fontsize=14, labelpad=10)
    plt.ylabel('ROC AUC Score (Accuracy)', fontsize=14, labelpad=10)
    plt.xlim(1, 52)
    plt.ylim(0.0, 1.0)
    
    # Legend outside plot
    plt.legend(bbox_to_anchor=(1.02, 1), loc='upper left', borderaxespad=0., fontsize=11)
    plt.tight_layout()
    plt.savefig(r'C:\Users\bootcamp\Desktop\sideprojects\research\12_month_performance.png', dpi=300, bbox_inches='tight')
    
    print(f"\nTotal Promotions: {promotions}")
    print(f"Total Degradations Prevented: {rejections}")
    print("Graph saved to workspace.")

if __name__ == "__main__":
    execute_simulation()
