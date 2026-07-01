import asyncio
import time
import os
from functools import lru_cache
from typing import List

import joblib
import numpy as np
from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI(title="Continuous Learning Titanic API")

# Global variables to hold model state
current_model = None
current_version = None
model_last_modified = 0

def load_model_if_changed():
    global current_model, current_version, model_last_modified
    model_path = "model.pkl"
    
    if not os.path.exists(model_path):
        return
        
    mtime = os.path.getmtime(model_path)
    if mtime > model_last_modified:
        try:
            payload = joblib.load(model_path)
            current_model = payload["model"]
            current_version = payload["version"]
            model_last_modified = mtime
            print(f"Loaded new model version: {current_version}")
        except Exception as e:
            print(f"Error loading model: {e}")

@app.on_event("startup")
async def startup_event():
    load_model_if_changed()

class TitanicPassenger(BaseModel):
    pclass: float
    sex: float # 1 for male, 0 for female
    age: float
    sibsp: float
    parch: float
    fare: float

    def to_array(self):
        return np.array([[self.pclass, self.sex, self.age, self.sibsp, self.parch, self.fare]])

    def to_tuple(self):
        return (self.pclass, self.sex, self.age, self.sibsp, self.parch, self.fare)

# Middleware or dependency to always check if model needs reloading
# In a real heavy traffic system, we'd do this in a background task, 
# but for demonstration we check file modification time on request.
@app.middleware("http")
async def check_model_update(request, call_next):
    load_model_if_changed()
    response = await call_next(request)
    return response

# -----------------------------------------------------------------
# 1. VERSION-KEYED CACHE (Best for Continuous Learning)
# -----------------------------------------------------------------
@lru_cache(maxsize=10000)
def predict_version_cached(version_id, passenger_tuple):
    """
    Notice the first argument is version_id. 
    If the model updates, version_id changes, making it a guaranteed cache MISS,
    immediately retrieving fresh predictions!
    """
    arr = np.array([passenger_tuple])
    return current_model.predict(arr)[0]

@app.post("/predict/cached")
async def predict_cached(passenger: TitanicPassenger):
    if current_model is None:
        return {"error": "Model not loaded yet."}
        
    start = time.perf_counter()
    
    # We pass current_version into the cache key!
    prediction = predict_version_cached(current_version, passenger.to_tuple())
    
    latency_ms = (time.perf_counter() - start) * 1000
    return {
        "survived": int(prediction),
        "method": "version_keyed_cache",
        "model_version": current_version,
        "latency_ms": latency_ms
    }

# -----------------------------------------------------------------
# 2. BATCH PROCESSOR (To catch Thundering Herds when cache resets)
# -----------------------------------------------------------------
BATCH_WINDOW_SECONDS = 0.01  # 10ms
MAX_BATCH_SIZE = 128

class BatchProcessor:
    def __init__(self):
        self.queue = []
        self.lock = asyncio.Lock()
        self.task_running = False

    async def predict(self, feature_array: np.ndarray, version: str):
        future = asyncio.Future()
        
        async with self.lock:
            # We also queue the version to ensure consistency
            self.queue.append((feature_array, future, version))
            if not self.task_running:
                self.task_running = True
                asyncio.create_task(self._process_queue())
        
        return await future

    async def _process_queue(self):
        await asyncio.sleep(BATCH_WINDOW_SECONDS)
        
        async with self.lock:
            items_to_process = self.queue[:MAX_BATCH_SIZE]
            self.queue = self.queue[MAX_BATCH_SIZE:]
            if len(self.queue) > 0:
                asyncio.create_task(self._process_queue())
            else:
                self.task_running = False

        if not items_to_process:
            return

        # Stack into one big batch
        X_batch = np.vstack([item[0] for item in items_to_process])
        futures = [item[1] for item in items_to_process]
        
        # Predict all at once using the globally active model
        predictions = current_model.predict(X_batch)
        
        for future, pred in zip(futures, predictions):
            future.set_result(pred)

batch_processor = BatchProcessor()

@app.post("/predict/batch")
async def predict_batch(passenger: TitanicPassenger):
    if current_model is None:
        return {"error": "Model not loaded yet."}
        
    start = time.perf_counter()
    arr = passenger.to_array()
    
    # Send through async batching
    prediction = await batch_processor.predict(arr, current_version)
    
    latency_ms = (time.perf_counter() - start) * 1000
    return {
        "survived": int(prediction),
        "method": "dynamic_batching",
        "model_version": current_version,
        "latency_ms": latency_ms
    }
