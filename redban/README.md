# redban

`movie` hammers old.reddit.com from a bunch of agents and gets 429s if you don't pace requests. This is a token bucket with state saved to disk so restarts don't reset your budget.

Default is 30 req/min with burst 5 — conservative on purpose. Run your curl/fetch after `--`:

```bash
cd redban
go build -o redban.exe .
./redban.exe -- curl -s "https://old.reddit.com/r/movies/search.json?q=underrated&restrict_sr=1"
./redban.exe -rpm 45 -state /tmp/reddit.tokens -- your-scraper-binary -agent drama
```

`./redban.exe` with no command just waits until a token is free and prints the balance.
