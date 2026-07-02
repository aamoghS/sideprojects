"""
Generates synthetic datasets that simulate different real-world fraud scenarios.
Each dataset tests a different distribution, drift pattern, and fraud rate.
"""
import numpy as np
import pandas as pd
from sklearn.preprocessing import LabelEncoder

RANDOM_SEED = 42
rng = np.random.default_rng(RANDOM_SEED)


def _save(df: pd.DataFrame, path: str):
    le = LabelEncoder()
    df['type_raw'] = df['type']
    df.to_csv(path, index=False)
    print(f"  Saved {len(df):,} rows -> {path}")


def generate_ecommerce_fraud(n: int = 70000) -> pd.DataFrame:
    """
    E-commerce chargeback fraud.
    - Low-value, high-volume transactions
    - Fraud bursts at night / weekend cycles
    - Drift: fraudsters start mimicking legitimate order amounts mid-year
    """
    steps = np.arange(n)
    txn_types = rng.choice(['PURCHASE', 'REFUND', 'CHARGEBACK', 'TRANSFER', 'PAYMENT'], size=n,
                           p=[0.60, 0.15, 0.05, 0.10, 0.10])
    amounts = np.where(txn_types == 'CHARGEBACK',
                       rng.exponential(scale=200, size=n),
                       rng.exponential(scale=80, size=n))
    old_bal = rng.uniform(0, 5000, size=n)
    new_bal = np.clip(old_bal - amounts, 0, None)
    dest_old = rng.uniform(0, 10000, size=n)
    dest_new = dest_old + amounts

    is_fraud = np.zeros(n, dtype=int)
    # Base fraud: chargebacks and high-value nighttime purchases
    fraud_mask = (txn_types == 'CHARGEBACK') & (rng.random(n) < 0.40)
    is_fraud[fraud_mask] = 1
    # Drift wave mid-year: fraudsters mimic normal purchase amounts
    drift_mask = (steps >= 35000) & (steps < 50000) & (txn_types == 'PURCHASE') & (amounts < 100)
    is_fraud[np.where(drift_mask)[0][rng.choice(drift_mask.sum(), size=int(drift_mask.sum() * 0.25), replace=False)]] = 1

    df = pd.DataFrame({
        'step': steps, 'type': txn_types, 'amount': amounts.round(2),
        'oldbalanceOrg': old_bal.round(2), 'newbalanceOrig': new_bal.round(2),
        'oldbalanceDest': dest_old.round(2), 'newbalanceDest': dest_new.round(2),
        'isFraud': is_fraud
    })
    return df.sort_values('step').reset_index(drop=True)


def generate_crypto_fraud(n: int = 70000) -> pd.DataFrame:
    """
    Crypto exchange wash trading / pump-and-dump.
    - Heavy-tail amount distribution (power law)
    - Fraud is correlated with large TRANSFER amounts
    - Drift: sudden volatility spikes create new fraud clusters at week ~30
    """
    steps = np.arange(n)
    txn_types = rng.choice(['TRANSFER', 'EXCHANGE', 'WITHDRAW', 'DEPOSIT', 'PAYMENT'], size=n,
                           p=[0.30, 0.25, 0.20, 0.15, 0.10])
    # Power-law amounts (Pareto) — more extreme than PaySim
    amounts = rng.pareto(a=1.5, size=n) * 500 + 10
    old_bal = rng.uniform(0, 100000, size=n)
    new_bal = np.clip(old_bal - amounts, 0, None)
    dest_old = rng.uniform(0, 200000, size=n)
    dest_new = dest_old + amounts

    is_fraud = np.zeros(n, dtype=int)
    # Base: large transfers are wash trades
    fraud_mask = (txn_types == 'TRANSFER') & (amounts > 5000) & (rng.random(n) < 0.50)
    is_fraud[fraud_mask] = 1
    # Drift: sudden pump-and-dump cluster at step ~30k
    drift_mask = (steps >= 30000) & (steps < 38000) & (txn_types == 'EXCHANGE') & (amounts > 1000)
    is_fraud[np.where(drift_mask)[0][rng.choice(drift_mask.sum(), size=int(drift_mask.sum() * 0.60), replace=False)]] = 1

    df = pd.DataFrame({
        'step': steps, 'type': txn_types, 'amount': amounts.round(2),
        'oldbalanceOrg': old_bal.round(2), 'newbalanceOrig': new_bal.round(2),
        'oldbalanceDest': dest_old.round(2), 'newbalanceDest': dest_new.round(2),
        'isFraud': is_fraud
    })
    return df.sort_values('step').reset_index(drop=True)


def generate_insurance_fraud(n: int = 70000) -> pd.DataFrame:
    """
    Insurance claim fraud.
    - Low fraud rate (~3%), very imbalanced
    - Clustered fraud rings (correlated claims in same period)
    - Drift: fraud ring emerges at week ~20 then again at week ~45
    """
    steps = np.arange(n)
    txn_types = rng.choice(['CLAIM', 'PAYMENT', 'REFUND', 'TRANSFER', 'DEBIT'], size=n,
                           p=[0.40, 0.30, 0.15, 0.10, 0.05])
    amounts = np.where(txn_types == 'CLAIM',
                       rng.lognormal(mean=8.5, sigma=1.2, size=n),
                       rng.lognormal(mean=6.0, sigma=0.8, size=n))
    old_bal = rng.uniform(1000, 50000, size=n)
    new_bal = np.clip(old_bal - amounts, 0, None)
    dest_old = rng.uniform(0, 20000, size=n)
    dest_new = dest_old + amounts

    is_fraud = np.zeros(n, dtype=int)
    # Base: small proportion of CLAIM types are inflated
    fraud_mask = (txn_types == 'CLAIM') & (amounts > 10000) & (rng.random(n) < 0.08)
    is_fraud[fraud_mask] = 1
    # Drift wave 1: organized ring at week ~20
    ring1_mask = (steps >= 20000) & (steps < 28000) & (txn_types == 'CLAIM')
    is_fraud[np.where(ring1_mask)[0][rng.choice(ring1_mask.sum(), size=int(ring1_mask.sum() * 0.35), replace=False)]] = 1
    # Drift wave 2: second ring at week ~45
    ring2_mask = (steps >= 45000) & (steps < 52000) & (txn_types == 'PAYMENT')
    is_fraud[np.where(ring2_mask)[0][rng.choice(ring2_mask.sum(), size=int(ring2_mask.sum() * 0.20), replace=False)]] = 1

    df = pd.DataFrame({
        'step': steps, 'type': txn_types, 'amount': amounts.round(2),
        'oldbalanceOrg': old_bal.round(2), 'newbalanceOrig': new_bal.round(2),
        'oldbalanceDest': dest_old.round(2), 'newbalanceDest': dest_new.round(2),
        'isFraud': is_fraud
    })
    return df.sort_values('step').reset_index(drop=True)


if __name__ == "__main__":
    print("Generating synthetic benchmark datasets...")

    df_ecom = generate_ecommerce_fraud()
    _save(df_ecom, "dataset_ecommerce.csv")
    print(f"  Fraud rate: {df_ecom['isFraud'].mean()*100:.1f}%")

    df_crypto = generate_crypto_fraud()
    _save(df_crypto, "dataset_crypto.csv")
    print(f"  Fraud rate: {df_crypto['isFraud'].mean()*100:.1f}%")

    df_ins = generate_insurance_fraud()
    _save(df_ins, "dataset_insurance.csv")
    print(f"  Fraud rate: {df_ins['isFraud'].mean()*100:.1f}%")

    print("\nAll datasets ready.")
