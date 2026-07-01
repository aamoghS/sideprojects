import requests
import time

url = "http://127.0.0.1:8000/predict/cached"
payload = {
    "pclass": 3,
    "sex": 1,
    "age": 22,
    "sibsp": 1,
    "parch": 0,
    "fare": 7.25
}

print("Pinging API every 1 second to observe seamless model version updates...\n")

for i in range(25):
    try:
        response = requests.post(url, json=payload)
        data = response.json()
        print(f"[{i:02d}] Prediction: {data.get('survived')} | "
              f"Model Version: {data.get('model_version')} | "
              f"Latency: {data.get('latency_ms'):.3f} ms")
    except Exception as e:
        print(f"[{i:02d}] Request failed: {e}")
    time.sleep(1)
print("\nTraffic simulation complete.")
