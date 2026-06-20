# Self-host your own proxy

Run **edgeproxy** on a VPS so movie-finder scrapes through your server IP instead of a third-party provider. You control the host, credentials, and firewall.

See also [PROXY-SETUP.md](./PROXY-SETUP.md) for docket/env wiring and [LOCAL-PROXY.md](../LOCAL-PROXY.md) for local dev with `localproxy`.

## Architecture

```
  [movie-finder]                    [your VPS]
  laptop / Docker        ------>    edgeproxy :8888
       |                                  |
       |  HTTP proxy + basic auth         |  outbound HTTP/HTTPS
       v                                  v
  proxy-docket.json              Reddit, Wikipedia, ...
  or -proxy / MOVIE_FINDER_PROXY
```

Movie-finder is the **client**. It sends requests to your VPS forward proxy (`http://user:pass@your-vps-ip:8888`). Edgeproxy accepts plain HTTP and HTTPS `CONNECT` tunnels, then dials the internet from the VPS egress IP. No third-party proxy vendor is involved.

## What you need

- A VPS (DigitalOcean, Hetzner, Linode, Vultr, etc.) with a public IPv4 address
- Docker and Docker Compose on the VPS (or build `edgeproxy` with Go)
- A strong username/password for proxy basic auth

## Deploy on a VPS

### 1. Copy the project (or just the proxy files)

On the VPS, clone or copy the `movie` repo. You only need:

- `deploy/proxy/Dockerfile`
- `docker-compose.proxy.yml`
- `go.mod`, `go.sum`, `cmd/edgeproxy`, `internal/proxy/server` (if building from source)

### 2. Set credentials

Create `.env` next to `docker-compose.proxy.yml`:

```env
PROXY_USER=your-username
PROXY_PASS=your-strong-password
PROXY_PORT=8888
```

Use a long random password. Anyone who can reach `:8888` without auth could relay traffic through your VPS.

### 3. Start the proxy

```bash
docker compose -f docker-compose.proxy.yml up -d --build
```

Verify health:

```bash
curl http://YOUR_VPS_IP:8888/health
# ok
```

Verify authenticated proxy (replace credentials and IP):

```bash
curl -x http://PROXY_USER:PROXY_PASS@YOUR_VPS_IP:8888 https://httpbin.org/ip
```

### 4. Lock down the firewall

Allow **only** port 8888 (or your chosen port) from IPs that need it — your home IP, office, or the host running movie-finder Docker.

Examples:

- **ufw (Ubuntu):** `ufw allow from YOUR_HOME_IP to any port 8888 proto tcp`
- **Hetzner / DO cloud firewall:** inbound TCP 8888 from your client IP only

Do **not** expose an open proxy without authentication. Edgeproxy refuses to start without `PROXY_USER` and `PROXY_PASS` unless `ALLOW_INSECURE=true` (local dev only).

## Point movie-finder at your proxy

### Option A — proxy docket (recommended)

Edit `config/proxy-docket.json`:

1. Open the `self-hosted` entry.
2. Replace `USER`, `PASS`, and `YOUR_VPS_IP` with real values.
3. Set `"enabled": true` and disable other entries.

```json
{
  "id": "self-hosted",
  "name": "My VPS proxy",
  "url": "http://scraper:long-random-pass@203.0.113.10:8888",
  "enabled": true,
  "provider": "self"
}
```

Test:

```bash
go run ./cmd/movie-finder -test-proxies
```

Run:

```bash
go run ./cmd/movie-finder
```

### Option B — CLI or environment

```bash
go run ./cmd/movie-finder -proxy http://USER:PASS@YOUR_VPS_IP:8888 -test-proxies
go run ./cmd/movie-finder -proxy http://USER:PASS@YOUR_VPS_IP:8888
```

Or in `.env`:

```env
MOVIE_FINDER_PROXY=http://USER:PASS@YOUR_VPS_IP:8888
```

### Docker movie-finder + remote proxy

On the machine running `docker compose up movie-finder`, set in `.env`:

```env
MOVIE_FINDER_PROXY=http://USER:PASS@YOUR_VPS_IP:8888
```

Or enable the `self-hosted` entry in the mounted `config/proxy-docket.json`.

## Run without Docker (binary on VPS)

```bash
export PROXY_USER=your-username
export PROXY_PASS=your-strong-password
export PROXY_ADDR=0.0.0.0:8888
go run ./cmd/edgeproxy
```

Or build and run:

```bash
go build -o edgeproxy ./cmd/edgeproxy
./edgeproxy
```

Use a process manager (systemd, supervisord) and bind behind your firewall.

## Local dev (same machine)

**Terminal 1** — proxy with auth:

```bash
set PROXY_USER=dev
set PROXY_PASS=secret
set ALLOW_INSECURE=true
go run ./cmd/edgeproxy
```

Or use `localproxy` (auth optional, binds `127.0.0.1` by default):

```bash
go run ./cmd/localproxy -user dev -pass secret
```

**Terminal 2** — test movie-finder:

```bash
go run ./cmd/movie-finder -proxy http://dev:secret@127.0.0.1:8888 -test-proxies
```

## Environment reference (edgeproxy)

| Variable | Default | Purpose |
|----------|---------|---------|
| `PROXY_ADDR` | `0.0.0.0:8888` | Listen address |
| `PROXY_USER` | (required) | Basic auth username |
| `PROXY_PASS` | (required) | Basic auth password |
| `ALLOW_INSECURE` | `false` | Set `true` to allow no auth (local dev only) |

Health check: `GET /health` returns `200` with body `ok` (no auth required).

## Security checklist

- [ ] Strong `PROXY_USER` / `PROXY_PASS`; never commit real values
- [ ] Firewall limits who can reach port 8888
- [ ] Do not set `ALLOW_INSECURE=true` on a public VPS
- [ ] Rotate credentials if a URL leaks (URLs embed the password)
- [ ] Monitor VPS bandwidth and logs for abuse

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `PROXY_USER and PROXY_PASS are required` | Set both in `.env` or export before starting edgeproxy |
| `407 Proxy Authentication Required` | Wrong user/pass in movie-finder URL |
| Connection timeout | Check VPS firewall, security groups, and that edgeproxy is listening on `0.0.0.0:8888` |
| Test OK but slow scrapes | VPS egress may be rate-limited by Reddit; try fewer workers |
