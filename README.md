# sideprojects

took some inspo from my friend jake paul

A monorepo of experiments — networking toys, ML scripts, and small tools. Each folder is its own module with its own dependencies; nothing here is production-ready.

## Go networking and infra

### [hotwire](hotwire/)

Latency-aware load balancer control plane. Backends (running `simhttp`) stream metrics to a central server over gRPC; the server scores each backend on latency and error rate, then pushes routing weights to proxies that pick backends probabilistically. Built to explore how a real LB control plane would work without standing up Envoy.

`go run ./cmd/hotwire demo` spins up the whole stack locally.

### [relay](relay/)

gRPC pub/sub CLI — publish messages to named topics, watch streams, or join bidirectional chat rooms. Mostly an excuse to get comfortable with protobuf streaming patterns.

`go build -o relay.exe ./cmd/relay && ./relay.exe serve`

### [simhttp](simhttp/)

Fake HTTP backend for load tests. Each instance serves configurable latency, jitter, and error rate, and exports p50/p99, error rate, inflight, and RPS snapshots. `hotwire` backends are thin wrappers around this.

Library only: `go test ./...`

### [plate](plate/)

Minimal VPS control plane. Run `plate serve` on a host; create VMs through an HTTP API or CLI. Docker provider for local dev, Proxmox provider for real KVM on a Linux box.

`./run.sh` in `plate/` for the Docker quick start.

### [ipv1](ipv1/)

Toy flat-address UDP protocol. Names like `1` and `host-3` instead of dotted quads; a JSON registry resolves them to endpoints. Joke "IPv1" for dev lab networks — plate's `ippool` still owns real IPv4.

`go build -o ipv1.exe ./cmd/ipv1 && ./ipv1.exe listen 1`

## Tools

### [slugcheck](slugcheck/)

Reads a site's `sitemap.xml`, follows redirects, and flags slug collisions — multiple sitemap URLs that land on the same final page — plus chains that bounce too many times. Complements `surf`'s on-page SEO checks with URL hygiene.

`go build -o slugcheck.exe . && ./slugcheck.exe https://example.com`

### [redban](redban/)

Token-bucket rate limiter with state on disk. Wrap Reddit fetches so `movie` agents don't burn through the unauthenticated quota in one run.

`go build -o redban.exe . && ./redban.exe -- curl -s "https://old.reddit.com/..."`

### [surf](surf/)

SEO analyzer CLI (`seotool`). Crawls a site, scores on-page SEO issues, and can batch-audit many URLs to JSONL with before/after diffing.

`go build -o seotool.exe ./cmd/seotool`

### [movie](movie/)

Multi-agent movie recommendation scraper. Each "agent" has genre-specific subreddits and seed picks; the tool searches Reddit threads, extracts titles, and enriches them with Wikipedia plots. Includes a local forward proxy and optional distributed orchestrator for running agents at scale.

See `movie/LOCAL-PROXY.md` for proxy setup. Run: `go run ./cmd/movie-finder`

### [ideas/simplijobs](ideas/simplijobs/)

CLI tracker for [SimplifyJobs](https://github.com/SimplifyJobs) internship and new-grad listings on GitHub. Remembers when you last checked and only surfaces new postings, with filters for company and location.

`go run . check internships`

### [bookRoom](bookRoom/)

Scripts for booking GT library study rooms through LibCal. Logs in with your GT credentials, checks availability for a configured room and time slot, and submits the booking.

Set `GT_USERNAME`, `GT_PASSWORD`, `GT_STUDENT_ID`, and `GT_USER_LAST_NAME`, then `go run .` in `gt-room-booking/`.

## Libraries

### [minstd](minstd/)

Small stdlib shims (`atomic`, `sync`, `http`, `net`, etc.) extracted while building `simhttp`. Each package covers one gap; not meant as a full stdlib replacement.

`go test ./...`

## Python / ML

### [optimization](optimization/)

RF and LightGBM hyperparameter tuning lab on the UCI Adult income dataset. Deliberately starts with weak baselines (~60% accuracy), then compares ten tuning methods (random search, Optuna, threshold sweeps, etc.) to see what actually moves the needle.

`python -m src.main all` after `pip install -r requirements.txt`

`calibrate.py` — given a saved `*.joblib` pipeline and a CSV, print binned predicted vs actual rates (Brier + ECE). Quick sanity check after tuning.

`python calibrate.py artifacts/baseline_rf.joblib adult.csv --target class`

### [research](research/)

ML experiments around concept drift and adaptive retraining. Synthetic fraud datasets (banking, ecommerce, crypto, insurance) with an incremental retraining pipeline that shadow-deploys candidate models and compares model families in benchmark shootouts.

See `train.py`, `run_benchmark.py`, `run_model_shootout.py`.

`ks_gaps.py` — two-sample KS distance between any two numeric columns in a CSV. Handy for spotting when a feature distribution drifts away from training data.

`python ks_gaps.py paysim.csv amount oldbalanceOrg`

## Misc

### [ideas/lol/gotero](ideas/lol/gotero/)

Go client for the Zotero group library API. Downloads PDFs from a named collection and its subcollections to a local folder.

`go run ./cmd/gotero -output-dir ./pdfs "Collection Name"`
