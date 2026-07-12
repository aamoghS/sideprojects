# simplijobs

Tracks new internship and new-grad postings from the [SimplifyJobs](https://github.com/SimplifyJobs) GitHub repos. Instead of manually diffing markdown tables in the browser, `check` fetches the current listings, compares against local state from your last run, and prints only what's new.

`status` shows when you last checked each source. `reset` clears that history if you want a fresh baseline.

## Build

```bash
cd ideas/simplijobs
go mod tidy
go build -o simplijobs.exe .
```

## Usage

```bash
simplijobs.exe check internships
simplijobs.exe check newgrad
simplijobs.exe check newgrad --all
simplijobs.exe status
simplijobs.exe reset
```

Filters: `--company`, `--location`, `--limit`, `--no-color`.
