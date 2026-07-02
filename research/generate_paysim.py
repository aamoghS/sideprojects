import pandas as pd
import numpy as np

def generate_paysim(num_rows=64000):
    np.random.seed(42)
    step = np.repeat(np.arange(1, (num_rows // 100) + 2), 100)[:num_rows]
    types = np.random.choice(['PAYMENT', 'TRANSFER', 'CASH_OUT', 'DEBIT', 'CASH_IN'], size=num_rows, p=[0.4, 0.1, 0.3, 0.05, 0.15])
    amount = np.random.exponential(scale=500, size=num_rows)
    oldbalanceOrg = np.random.exponential(scale=2000, size=num_rows)
    newbalanceOrig = np.where(np.isin(types, ['CASH_OUT', 'PAYMENT', 'TRANSFER', 'DEBIT']),
                              np.maximum(oldbalanceOrg - amount, 0),
                              oldbalanceOrg + amount)
    oldbalanceDest = np.random.exponential(scale=5000, size=num_rows)
    newbalanceDest = np.where(np.isin(types, ['CASH_IN', 'PAYMENT']),
                              np.maximum(oldbalanceDest - amount, 0),
                              oldbalanceDest + amount)

    df = pd.DataFrame({
        'step': step,
        'type': types,
        'amount': amount,
        'oldbalanceOrg': oldbalanceOrg,
        'newbalanceOrig': newbalanceOrig,
        'oldbalanceDest': oldbalanceDest,
        'newbalanceDest': newbalanceDest
    })

    # Base fraud logic (Transfer & Cash_out of large amounts mostly)
    fraud_mask = (df['type'].isin(['TRANSFER', 'CASH_OUT'])) & (df['amount'] > 4000)
    df['isFraud'] = 0
    df.loc[fraud_mask, 'isFraud'] = np.random.choice([0, 1], size=fraud_mask.sum(), p=[0.5, 0.5])
    
    # Save base dataset
    df.to_csv("paysim.csv", index=False)

if __name__ == "__main__":
    generate_paysim()
