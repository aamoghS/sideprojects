"""
Deep Learning (LSTM) vs CatBoost Backtester for Market Regime.
Predicts forward returns and simulates a simple trading strategy (PnL).
"""
import os
import warnings
import numpy as np
import pandas as pd
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import torch
import torch.nn as nn
from torch.utils.data import DataLoader, TensorDataset
from catboost import CatBoostClassifier

warnings.filterwarnings('ignore')

# ── Config ────────────────────────────────────────────────────────────────────
DATA_PATH = 'dataset_market.csv'
OUT_PNG   = 'market_pnl_backtest.png'
SEQ_LEN   = 10     # Sequence length for LSTM
TRAIN_END = 20000  # Train on first 20k steps
TEST_END  = 40000  # Backtest on next 20k steps

# ── Data Loading & Scaling ────────────────────────────────────────────────────
print("Loading Market Dataset...")
df = pd.read_csv(DATA_PATH, nrows=TEST_END).sort_values('step').reset_index(drop=True)

# Select numeric features
exclude = {'step', 'type', 'isFraud'}
feats = [c for c in df.columns if c not in exclude and pd.api.types.is_numeric_dtype(df[c])]
print(f"Using {len(feats)} features.")

# Scale features (Standardization based on train set)
train_df = df.iloc[:TRAIN_END].copy()
test_df  = df.iloc[TRAIN_END:TEST_END].copy()

means = train_df[feats].mean()
stds  = train_df[feats].std() + 1e-9

train_scaled = (train_df[feats] - means) / stds
test_scaled  = (test_df[feats] - means) / stds

# ── Sequence Generation for LSTM ──────────────────────────────────────────────
def create_sequences(data, labels, seq_len):
    xs, ys = [], []
    for i in range(len(data) - seq_len):
        xs.append(data.iloc[i:(i + seq_len)].values)
        ys.append(labels.iloc[i + seq_len - 1])
    return np.array(xs), np.array(ys)

print("Building Sequences...")
X_train_seq, y_train_seq = create_sequences(train_scaled, train_df['isFraud'], SEQ_LEN)
X_test_seq, y_test_seq   = create_sequences(test_scaled, test_df['isFraud'], SEQ_LEN)

# Convert to PyTorch tensors
X_train_t = torch.tensor(X_train_seq, dtype=torch.float32)
y_train_t = torch.tensor(y_train_seq, dtype=torch.float32).unsqueeze(1)
X_test_t  = torch.tensor(X_test_seq, dtype=torch.float32)
y_test_t  = torch.tensor(y_test_seq, dtype=torch.float32).unsqueeze(1)

train_loader = DataLoader(TensorDataset(X_train_t, y_train_t), batch_size=256, shuffle=True)

# ── LSTM Model ────────────────────────────────────────────────────────────────
class MarketLSTM(nn.Module):
    def __init__(self, input_dim, hidden_dim=64, num_layers=2):
        super().__init__()
        self.lstm = nn.LSTM(input_dim, hidden_dim, num_layers, batch_first=True, dropout=0.2)
        self.fc = nn.Linear(hidden_dim, 1)
        self.sigmoid = nn.Sigmoid()
        
    def forward(self, x):
        out, _ = self.lstm(x)
        out = self.fc(out[:, -1, :])  # Take last time step
        return self.sigmoid(out)

print("Training LSTM Model...")
device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
model = MarketLSTM(input_dim=len(feats)).to(device)
criterion = nn.BCELoss()
optimizer = torch.optim.Adam(model.parameters(), lr=0.001)

model.train()
for epoch in range(5):  # Short training for quick feedback
    total_loss = 0
    for batch_X, batch_y in train_loader:
        batch_X, batch_y = batch_X.to(device), batch_y.to(device)
        optimizer.zero_grad()
        preds = model(batch_X)
        loss = criterion(preds, batch_y)
        loss.backward()
        optimizer.step()
        total_loss += loss.item()
    print(f"  Epoch {epoch+1} Loss: {total_loss/len(train_loader):.4f}")

model.eval()
with torch.no_grad():
    lstm_preds = model(X_test_t.to(device)).cpu().numpy().flatten()

# Pad predictions to match test_df length (due to seq_len)
lstm_preds_padded = np.concatenate([np.full(SEQ_LEN, 0.5), lstm_preds])

# ── CatBoost Baseline ─────────────────────────────────────────────────────────
print("Training CatBoost Baseline...")
cb = CatBoostClassifier(iterations=100, depth=6, learning_rate=0.05, verbose=False, random_seed=42)
cb.fit(train_scaled, train_df['isFraud'])
cb_preds = cb.predict_proba(test_scaled)[:, 1]

# ── PnL Backtesting ───────────────────────────────────────────────────────────
print("Running Backtest...")
# We trade based on predictions:
# If pred > 0.55 -> Go Long (+1)
# If pred < 0.45 -> Go Short (-1)
# Else -> Flat (0)

# Calculate step-by-step returns (price difference from t to t+1)
# amount = price. Return = (Price_{t+1} - Price_t) / Price_t
prices = test_df['amount'].values
actual_returns = np.zeros_like(prices)
actual_returns[:-1] = (prices[1:] - prices[:-1]) / prices[:-1]

def simulate_pnl(preds, returns, threshold_long=0.55, threshold_short=0.45):
    positions = np.zeros_like(preds)
    positions[preds > threshold_long] = 1.0
    positions[preds < threshold_short] = -1.0
    
    # PnL is position * actual forward return
    strategy_returns = positions * returns
    # Cumulative compound return (starting with $10,000)
    equity = 10000 * np.cumprod(1 + strategy_returns)
    return equity

# Buy and hold equity
bh_equity = 10000 * np.cumprod(1 + actual_returns)

lstm_equity = simulate_pnl(lstm_preds_padded, actual_returns)
cb_equity   = simulate_pnl(cb_preds, actual_returns)

# ── Plotting ──────────────────────────────────────────────────────────────────
fig = plt.figure(figsize=(14, 7))
fig.patch.set_facecolor('#0D1117')
ax = fig.add_subplot(111)
ax.set_facecolor('#161B22')
ax.tick_params(colors='#8B949E')
for sp in ax.spines.values():
    sp.set_edgecolor('#30363D')

steps = np.arange(len(bh_equity))
ax.plot(steps, bh_equity, color='#888888', label='Buy & Hold', alpha=0.7, lw=2)
ax.plot(steps, cb_equity, color='#A8D5A2', label=f'CatBoost (End: ${cb_equity[-1]:,.2f})', lw=2)
ax.plot(steps, lstm_equity, color='#4C9BE8', label=f'LSTM (End: ${lstm_equity[-1]:,.2f})', lw=2)

ax.set_title('Deep Learning vs CatBoost: Market Regime PnL Backtest', color='#E6EDF3', fontsize=14, fontweight='bold', pad=15)
ax.set_ylabel('Portfolio Value ($)', color='#8B949E', fontsize=11)
ax.set_xlabel('Time Steps', color='#8B949E', fontsize=11)
ax.legend(facecolor='#161B22', labelcolor='#E6EDF3', edgecolor='#30363D')
ax.grid(color='#30363D', linestyle='--', alpha=0.5)

plt.savefig(OUT_PNG, dpi=150, bbox_inches='tight', facecolor=fig.get_facecolor())
print(f"Backtest saved to {OUT_PNG}")
