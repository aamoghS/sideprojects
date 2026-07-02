import os
import time
import logging
import hashlib
from functools import lru_cache
from typing import Dict, Any

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import lightgbm as lgb
import joblib
import pandas as pd
import redis

logging.basicConfig(level=logging.INFO, format='%(asctime)s [%(levelname)s] %(message)s')
logger = logging.getLogger(__name__)

app = FastAPI(title="Fraud Detection API")

MODEL_PATH = "primary_model.txt"
ENCODER_PATH = "label_encoder.joblib"
CACHE_TTL = 3600

# Global state
primary_model: lgb.Booster = None
label_encoder = None
redis_client = redis.Redis(host='localhost', port=6379, db=0, decode_responses=True)

class Transaction(BaseModel):
    step: int
    type: str
    amount: float
    oldbalanceOrg: float
    newbalanceOrig: float
    oldbalanceDest: float
    newbalanceDest: float

@app.on_event("startup")
def initialize_services():
    global primary_model, label_encoder
    if not os.path.exists(MODEL_PATH) or not os.path.exists(ENCODER_PATH):
        logger.error("Required model assets missing at startup.")
        raise RuntimeError("Model files not found.")

    primary_model = lgb.Booster(model_file=MODEL_PATH)
    label_encoder = joblib.load(ENCODER_PATH)

    try:
        redis_client.ping()
        logger.info("Redis connection established.")
    except redis.exceptions.ConnectionError:
        logger.warning("Redis is unavailable. Caching disabled.")

def get_cache_key(txn: Transaction) -> str:
    raw_str = f"{txn.step}:{txn.type}:{txn.amount}:{txn.oldbalanceOrg}:{txn.newbalanceOrig}:{txn.oldbalanceDest}:{txn.newbalanceDest}"
    return f"fraud_pred:{hashlib.sha256(raw_str.encode()).hexdigest()}"

def compute_prediction(txn: Transaction) -> float:
    type_idx = label_encoder.transform([txn.type])[0] if txn.type in label_encoder.classes_ else 0
    features = pd.DataFrame([{
        'step': txn.step,
        'type': type_idx,
        'amount': txn.amount,
        'oldbalanceOrg': txn.oldbalanceOrg,
        'newbalanceOrig': txn.newbalanceOrig,
        'oldbalanceDest': txn.oldbalanceDest,
        'newbalanceDest': txn.newbalanceDest
    }])
    return float(primary_model.predict(features)[0])

@app.post("/predict")
def predict(txn: Transaction) -> Dict[str, Any]:
    start_time = time.time()
    cache_key = get_cache_key(txn)

    try:
        cached_val = redis_client.get(cache_key)
        if cached_val is not None:
            prob = float(cached_val)
            cache_hit = True
        else:
            prob = compute_prediction(txn)
            redis_client.setex(cache_key, CACHE_TTL, str(prob))
            cache_hit = False
    except redis.exceptions.ConnectionError:
        prob = compute_prediction(txn)
        cache_hit = False

    return {
        "is_fraud": bool(prob > 0.05),
        "probability": round(prob, 4),
        "latency_ms": round((time.time() - start_time) * 1000, 2),
        "cache_hit": cache_hit
    }

@app.post("/reload_model")
def reload_model() -> Dict[str, str]:
    global primary_model
    try:
        primary_model = lgb.Booster(model_file=MODEL_PATH)
        redis_client.flushdb()
        logger.info("Model reloaded and cache flushed.")
        return {"status": "ok"}
    except Exception as e:
        logger.error(f"Reload failed: {str(e)}")
        raise HTTPException(status_code=500, detail="Model reload failed")
