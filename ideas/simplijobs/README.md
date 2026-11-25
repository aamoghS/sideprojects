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

## System design

How a request or a run moves through the code, then the choices behind that shape. In the decision charts, a **solid** box is the route this repo takes and a **dashed** box is the alternative it turns down; the edge label is the tradeoff.

### New rows only

check fetches SimplifyJobs listings, compares them to the last local snapshot, and prints the new ones. Company and location filters sit on that diff. status shows the last check. reset forgets it.

```mermaid
flowchart TD
  classDef chosen fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px,color:#1b5e20
  src["SimplifyJobs listings"]
  last["Last local snapshot"]
  diff["New rows"]
  filt["Company and location filters"]
  src --> diff
  last --> diff
  diff --> filt
  class src,last,diff,filt chosen
```

### A diff, stored locally

The list is long and most of it was there yesterday. Remembering the last check is the product. There is no account. A changed title looks new. A listing that disappeared is not the headline.

```mermaid
flowchart TD
  classDef chosen fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px,color:#1b5e20
  classDef rejected fill:#f5f5f5,stroke:#9e9e9e,stroke-width:1px,stroke-dasharray:5 5,color:#616161
  q{What to print}
  q -->|"rows since the last check"| diff["The diff"]
  q -->|"the whole list again"| allrows["The full listing"]
  state{Who remembers the check}
  state -->|"a local file, status and reset"| local["No account"]
  state -->|"a login"| acct["An account"]
  class diff,local chosen
  class allrows,acct rejected
```

Longer write-up: [aamogh.vercel.app/projects/simplijobs](https://aamogh.vercel.app/projects/simplijobs.html)
