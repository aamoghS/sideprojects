# surf

CLI for crawling sites and auditing on-page SEO. Given a URL it fetches the page, checks title/meta/heading structure, link hygiene, and similar signals, then scores what needs fixing.

The `audit` and `batch` commands crawl multiple pages and write results to JSONL. `plan` turns audit output into a prioritized fix list; `diff` compares before/after audits so you can see what changed after deploying fixes.

## Build

```bash
cd surf
go mod tidy
go build -o seotool.exe ./cmd/seotool
```

## Commands

```bash
./seotool.exe analyze --url https://example.com
./seotool.exe audit --url https://example.com --depth 2 --output audit.jsonl
./seotool.exe batch --urls-file config/urls.example.txt --output results.jsonl
./seotool.exe plan --input audit.jsonl
./seotool.exe diff --before audit-before.jsonl --after audit-after.jsonl
```

Tests: `go test ./...`
