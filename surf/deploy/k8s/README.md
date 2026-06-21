# SEO batch crawl on Kubernetes

Reuses the `movie-proxy` Deployment in namespace `crawl`. Each indexed pod analyzes a shard of URLs and writes JSONL.

## Run

Apply movie proxy stack first (`movie/deploy/k8s`), then:

```powershell
kubectl apply -f deploy/k8s/job-seo-batch.yaml
kubectl logs -n crawl -l job-name=seo-audit
kubectl cp crawl/seo-audit-0:/out/results.jsonl ./shard-0.jsonl
```

## Local

```powershell
$env:HTTP_PROXY="http://127.0.0.1:8888"
go run ./cmd/seotool batch --urls-file config/urls.example.txt --output results.jsonl
```

## vs movie agents

| | movie | surf batch |
|---|-------|------------|
| Unit of work | agent (genre/queries) | URL |
| Sharding | `JOB_COMPLETION_INDEX` → agent | index → URL lines |
| Output | JSON per agent | JSONL per URL |
| Proxy | same `movie-proxy` Service | same |

This is for **SEO audits** (meta tags, headings, score). Not for manipulating rankings.

## Audit workflow (improve rankings legitimately)

1. **Baseline audit** — crawl the site and record issues + fixes:

```powershell
go run ./cmd/seotool audit --url https://yoursite.com --depth 2 --output audit-before.jsonl --plan fixes.txt
```

2. **Apply fixes** from `fixes.txt` (titles, meta, H1, canonical, alt text, etc.) in your CMS or templates.

3. **Re-audit** after deploy:

```powershell
go run ./cmd/seotool audit --url https://yoursite.com --depth 2 --output audit-after.jsonl
```

4. **Measure progress**:

```powershell
go run ./cmd/seotool diff --before audit-before.jsonl --after audit-after.jsonl
```

`plan` works on any JSONL from `audit` or `batch`:

```powershell
go run ./cmd/seotool plan --input results.jsonl
```
