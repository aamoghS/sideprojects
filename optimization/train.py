import time
import joblib
import pandas as pd
import lightgbm as lgb
from sklearn.datasets import fetch_openml
from sklearn.model_selection import train_test_split
from sklearn.metrics import accuracy_score, roc_auc_score
import os

from preprocess import preprocess_features

print("Fetching Titanic Dataset...")
titanic = fetch_openml('titanic', version=1, as_frame=True)
df = titanic.frame

# We will sample slightly to simulate the "continuous learning" data drift
df = df.sample(frac=0.8, random_state=int(time.time()) % 10000)
y = df['survived'].astype(int)
X_raw = df.drop(columns=['survived'])

print("Preprocessing features...")
X = preprocess_features(X_raw, is_training=True)

# Best Practice: Train/Validation Split for Early Stopping and Evaluation
X_train, X_val, y_train, y_val = train_test_split(X, y, test_size=0.2, random_state=42)

print(f"Training on {len(X_train)} rows, Validating on {len(X_val)} rows...")

# Best Practice: Hyperparameter tuning (basic setup for LightGBM)
model = lgb.LGBMClassifier(
    n_estimators=500, # Set high, early stopping will halt it
    learning_rate=0.05,
    max_depth=5,
    num_leaves=31,
    random_state=42,
    class_weight='balanced'
)

# Best Practice: Early stopping to prevent overfitting
callbacks = [
    lgb.early_stopping(stopping_rounds=20, verbose=True),
]

model.fit(
    X_train, y_train,
    eval_set=[(X_val, y_val)],
    callbacks=callbacks
)

# Evaluate the model
val_preds = model.predict(X_val)
val_probs = model.predict_proba(X_val)[:, 1]

acc = accuracy_score(y_val, val_preds)
auc = roc_auc_score(y_val, val_probs)

print(f"\n--- Model Evaluation ---")
print(f"Accuracy: {acc:.4f}")
print(f"ROC-AUC:  {auc:.4f}")
print(f"------------------------\n")

# Display feature importances
importance = pd.DataFrame({
    'Feature': X.columns,
    'Importance': model.feature_importances_
}).sort_values(by='Importance', ascending=False)
print("Top 5 Feature Importances:")
print(importance.head(5))

# Create a version timestamp
version_id = str(int(time.time()))
payload = {
    "model": model,
    "version": version_id,
    "features": list(X.columns)
}

temp_file = "model.pkl.tmp"
joblib.dump(payload, temp_file)
os.replace(temp_file, "model.pkl")

print(f"\nModel version {version_id} saved successfully!")
