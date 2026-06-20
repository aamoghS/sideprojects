# External proxy setup

Movie-finder can route Reddit and Wikipedia requests through **your** proxy provider so traffic does not use your home or server IP directly. This project does **not** host or sell proxies — you must bring credentials from a provider or run your own gateway on a VPS.

See also [LOCAL-PROXY.md](../LOCAL-PROXY.md) for a local dev forward proxy (`cmd/localproxy`).

## What we cannot do for you

- Provision datacenter or residential IPs
- Create paid accounts (Webshare, Bright Data, IPRoyal, Oxylabs, etc.)
- Scrape or enable free public proxy lists

You sign up with a provider (or self-host), paste **your** URL into `config/proxy-docket.json` or `.env`, then test with `-test-proxies`.

## Quick start

1. **Choose a provider** (examples):
   - [Webshare](https://www.webshare.io/) — HTTP residential/datacenter
   - [Bright Data](https://brightdata.com/) — HTTP/SOCKS5, many products
   - [IPRoyal](https://iproyal.com/) — SOCKS5 and HTTP
   - [Oxylabs](https://oxylabs.io/) — HTTP datacenter/residential
   - **Self-host** — run `edgeproxy` on a VPS ([SELF-HOST-PROXY.md](./SELF-HOST-PROXY.md)) or `localproxy` locally ([LOCAL-PROXY.md](../LOCAL-PROXY.md))

2. **Copy credentials from the provider dashboard** (username, password, gateway host, port).

3. **Edit `config/proxy-docket.json`**:
   - Pick a template entry (`http-residential`, `http-datacenter`, or `socks5-residential`) or add your own object under `proxies`.
   - Replace `USER`, `PASS`, `YOUR-PROXY-HOST`, and the port with real values.
   - Set `"enabled": true` on the entry you want to use.
   - Leave all other entries `"enabled": false` (including `local` unless you are running `localproxy` on the same machine).

4. **Test connectivity**:

   ```bash
   go run ./cmd/movie-finder -test-proxies
   ```

   Expected when nothing is enabled yet:

   ```text
   Proxy test failed: no enabled proxies in docket — set enabled: true and add your proxy url
   ```

   After a real proxy is enabled and configured, you should see `OK` lines for each enabled entry.

5. **Run the app** (docket is used by default):

   ```bash
   go run ./cmd/movie-finder
   ```

## Proxy URL formats

| Scheme   | Example |
|----------|---------|
| HTTP     | `http://username:password@gate.provider.com:8080` |
| HTTPS    | `https://username:password@gate.provider.com:443` |
| SOCKS5   | `socks5://username:password@gate.provider.com:1080` |

Special characters in username or password must be [URL-encoded](https://en.wikipedia.org/wiki/Percent-encoding) (e.g. `@` → `%40`).

Provider-specific examples (replace with your values):

```text
http://customer-USER:PASS@pr.oxylabs.io:7777
http://USER:PASS@p.webshare.io:80
socks5://USER:PASS@geo.iproyal.com:12321
```

## Docket fields

File: `config/proxy-docket.json`

| Field | Purpose |
|-------|---------|
| `rotation` | `"round_robin"` — assign enabled proxies to agents in order (agent 0 → proxy 0, agent 1 → proxy 1, wraps around). |
| `default_proxy` | Optional proxy `id`. When set, **every agent** uses that single enabled entry (ignores round-robin). Leave `""` to rotate. |
| `proxies[].id` | Stable id (referenced by `default_proxy`). |
| `proxies[].name` | Human label in logs. |
| `proxies[].url` | Full proxy URL (`http://`, `https://`, or `socks5://`). |
| `proxies[].enabled` | Must be `true` for the entry to be used or tested. |
| `proxies[].provider` | Label only (e.g. `webshare`, `self`) — not sent to the network. |
| `proxies[].notes` | Reminders for your team. |
| `proxies[].tags` | Optional labels (`residential`, `socks5`, etc.). |

### Round-robin example

Three agents, two enabled proxies (`proxy-a`, `proxy-b`):

| Agent index | Proxy |
|-------------|-------|
| 0 | proxy-a |
| 1 | proxy-b |
| 2 | proxy-a |

Set `"default_proxy": "proxy-a"` to force all agents through one endpoint.

## Priority order (single proxy override)

When resolving which proxy an agent uses:

1. Per-agent `proxy` in `agents.json`
2. Proxy docket (`default_proxy` or round-robin)
3. `proxies[]` pool in `agents.json`
4. Global `proxy` in `agents.json`
5. `-proxy` flag
6. `MOVIE_FINDER_PROXY` env var
7. `HTTPS_PROXY` / `HTTP_PROXY`

First non-empty, valid URL wins.

## Environment variable (alternative to docket)

Copy `.env.example` to `.env` and set one URL for all agents:

```bash
cp .env.example .env
```

```env
MOVIE_FINDER_PROXY=http://username:password@gate.provider.com:8080
```

Leave empty if you rely on the docket only. `MOVIE_FINDER_PROXY` is used when the docket has no enabled proxies or when higher-priority sources are empty.

## Docker

```bash
cp .env.example .env
# Edit .env — set MOVIE_FINDER_PROXY and/or edit config/proxy-docket.json on the host
docker compose up --build
```

`docker-compose.yml` mounts `config/proxy-docket.json` read-only and passes `MOVIE_FINDER_PROXY` from `.env`. Enable real proxies in the docket **before** expecting outbound traffic to use them.

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `no enabled proxies in docket` | Set `enabled: true` on at least one entry with a real URL. |
| `still has placeholder values` | Replace `USER`, `PASS`, `YOUR-PROXY-HOST` in the URL. |
| `request failed` / timeout | Check host, port, firewall, and provider IP allowlist. |
| Reddit `403` / `429` in test | Often still OK — test treats 403/429 as reachable; scraping may still be rate-limited. |
| Direct IP / bans | Confirm banner shows docket proxy count > 0 or `MOVIE_FINDER_PROXY` is set. |

## Security

- Do not commit real credentials. Keep `.env` local; use placeholders in `proxy-docket.json` until you deploy.
- Rotate passwords if a URL is leaked — proxy URLs embed credentials.
