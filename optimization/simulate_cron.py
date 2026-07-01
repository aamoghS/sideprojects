import subprocess
import time

print("Starting Continuous Learning Cron Job Simulator...")
print("This script will retrain the model every 10 seconds.")
print("Press Ctrl+C to stop.\n")

while True:
    print(f"[{time.strftime('%H:%M:%S')}] Triggering model retraining...")
    # Run the training script in a subprocess using the venv python
    subprocess.run([".\\venv\\Scripts\\python.exe", "train.py"])
    print(f"[{time.strftime('%H:%M:%S')}] Waiting 10 seconds before next training cycle...\n")
    time.sleep(10)
