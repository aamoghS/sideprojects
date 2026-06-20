# Local HTTP forward proxy

A small Go proxy in `cmd/localproxy` that routes outbound HTTP/HTTPS through your machine. Use it with movie-finder so scraping traffic goes through a local proxy instead of your home IP or a paid service.

## Forward proxy vs reverse proxy

| | Forward proxy | Reverse proxy |
|---|---|---|
| **Sits in front of** | Clients (your app, curl, browser) | Servers (your web app) |
| **Client knows about it?** | Yes — you configure `-proxy` or `HTTP_PROXY` | No — clients hit a public URL |
| **Typical use** | Hide client IP, corporate egress, dev testing | Load balancing, TLS termination, caching |

Movie-finder is a **client** that fetches Reddit/Wikipedia. A **forward proxy** accepts requests from movie-finder and forwards them to the internet on your behalf.

### How this proxy works

1. **Plain HTTP** — The client sends a full URL (`GET http://example.com/path HTTP/1.1`). The proxy opens a connection to the target, relays the request, and returns the response.

2. **HTTPS (CONNECT)** — The client sends `CONNECT reddit.com:443 HTTP/1.1`. The proxy dials `reddit.com:443`, replies `200 Connection Established`, then tunnels encrypted bytes between client and server without decrypting TLS.

3. **Optional basic auth** — If you set `-user` / `-pass`, clients must send `Proxy-Authorization: Basic ...`. Off by default for local dev.

## Run the proxy

From the `movie` module root:

```bash
go run ./cmd/localproxy
```

Defaults to `127.0.0.1:8888`. Customize:

```bash
go run ./cmd/localproxy -addr 127.0.0.1:8888
go run ./cmd/localproxy -user dev -pass secret   # enable basic auth
```

Environment variables:

- `LOCAL_PROXY_ADDR` — listen address (overridden by `-addr`)
- `LOCAL_PROXY_USER` / `LOCAL_PROXY_PASS` — basic auth credentials

## Point movie-finder at the proxy

**Terminal 1** — start the proxy:

```bash
go run ./cmd/localproxy
```

**Terminal 2** — run movie-finder through it:

```bash
go run ./cmd/movie-finder -proxy http://127.0.0.1:8888
```

Or with an environment variable:

```bash
set MOVIE_FINDER_PROXY=http://127.0.0.1:8888
go run ./cmd/movie-finder
```

With basic auth enabled on the proxy:

```bash
go run ./cmd/movie-finder -proxy http://dev:secret@127.0.0.1:8888
```

Priority order in movie-finder: per-agent proxy → proxy docket → config `proxies` → config `proxy` → `-proxy` flag → `MOVIE_FINDER_PROXY` → `HTTPS_PROXY` / `HTTP_PROXY`.

## Test with curl

Start the proxy, then in another terminal:

```bash
curl -x http://127.0.0.1:8888 https://httpbin.org/ip
```

You should see JSON with the IP your proxy machine uses. The proxy terminal logs lines like:

```
CONNECT httpbin.org:443
```

For plain HTTP:

```bash
curl -x http://127.0.0.1:8888 http://httpbin.org/get
```

With basic auth:

```bash
go run ./cmd/localproxy -user dev -pass secret
curl -x http://dev:secret@127.0.0.1:8888 https://httpbin.org/ip
```

## Notes

- This is a **development / learning** proxy, not hardened for production or untrusted networks.
- Traffic still exits from **your machine's IP** — you are not hiding behind another datacenter unless you chain to an upstream proxy.
- For SOCKS5 or rotating paid proxies, use `proxy-docket.json` or `-proxy socks5://...` instead.
