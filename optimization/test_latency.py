import asyncio
import time
import httpx
import statistics

# A dummy payload matching the 20 features
PAYLOAD = {f"f{i}": 0.5 for i in range(1, 21)}

async def load_test_endpoint(client, url, num_requests=1000):
    start_time = time.perf_counter()
    
    # Fire off all requests concurrently
    tasks = [client.post(url, json=PAYLOAD) for _ in range(num_requests)]
    responses = await asyncio.gather(*tasks)
    
    total_time = time.perf_counter() - start_time
    
    # Extract latency reported by the server
    latencies = [res.json().get("latency_ms", 0) for res in responses if res.status_code == 200]
    
    if not latencies:
        print(f"Failed to get successful responses for {url}")
        return
        
    avg_latency = statistics.mean(latencies)
    throughput = num_requests / total_time
    
    print(f"Endpoint: {url}")
    print(f"  Total Time for {num_requests} reqs: {total_time:.2f}s")
    print(f"  Throughput: {throughput:.2f} req/s")
    print(f"  Avg Server Latency: {avg_latency:.4f} ms")
    print("-" * 40)

async def main():
    base_url = "http://127.0.0.1:8000/predict"
    endpoints = ["/raw", "/lru", "/ttl", "/batch"]
    
    # Give the server a moment to spin up
    print("Waiting for server to start...")
    await asyncio.sleep(2)
    
    async with httpx.AsyncClient(timeout=30.0) as client:
        # Warmup the model
        print("Warming up endpoints...")
        for ep in endpoints:
            await client.post(f"{base_url}{ep}", json=PAYLOAD)
            
        print("\nStarting Load Test (1000 concurrent requests each)...\n")
        
        for ep in endpoints:
            await load_test_endpoint(client, f"{base_url}{ep}", 1000)

if __name__ == "__main__":
    asyncio.run(main())
