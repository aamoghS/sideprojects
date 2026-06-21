# Kubernetes crawl stack

Proxy pods plus indexed agent jobs. Same layout works for SEO batch crawls in `surf/deploy/k8s`.

## Build images

From repo root:

```powershell
cd movie
docker build -t movie-finder:latest .
docker build -f deploy/proxy/Dockerfile -t movie-proxy:latest .

cd ..\surf
docker build -t seotool:latest .
```

Load into kind/minikube if local:

```powershell
kind load docker-image movie-finder:latest movie-proxy:latest seotool:latest
```

## Movie crawl

1. Edit `secret-proxy.yaml` if you want auth on the proxy (optional when `ALLOW_INSECURE=true` in-cluster).
2. Replace `configmap-agents.yaml` with your full `config/agents.json` if needed.
3. Set `completions` and `parallelism` in `job-movie-crawl.yaml` to match agent count.

```powershell
kubectl apply -f deploy/k8s/namespace.yaml
kubectl apply -f deploy/k8s/secret-proxy.yaml
kubectl apply -f deploy/k8s/configmap-agents.yaml
kubectl apply -f deploy/k8s/proxy.yaml
kubectl apply -f deploy/k8s/job-movie-crawl.yaml
```

Each indexed pod runs one agent (`JOB_COMPLETION_INDEX`). Results land in `/out/result.json` on the pod:

```powershell
kubectl logs job/movie-crawl-0 -n crawl
kubectl cp crawl/movie-crawl-0:/out/result.json ./drama.json
```

## Env vars

| Var | Purpose |
|-----|---------|
| `HTTP_PROXY` / `HTTPS_PROXY` | Egress via in-cluster proxy Service |
| `MOVIE_PROXIES` | Comma-separated list, rotated per agent |
| `JOB_COMPLETION_INDEX` | Set by indexed Job; picks one agent |
| `-agent drama` | Run a single agent locally |

## SEO batch (surf)

See `surf/deploy/k8s/README.md`. Uses the same proxy Deployment and `seotool batch` with URL sharding.
