package app

import (
	"flag"
	"os"
	"time"
)

type Options struct {
	Config     string
	Sequential bool
	Offline    bool
	Workers    int
	Proxy      string
	Docket     string
	TestProxies bool
	Timeout    time.Duration
}

func ParseFlags() Options {
	config := flag.String("config", "config/agents.json", "Path to your agents config file")
	sequential := flag.Bool("sequential", false, "Run agents one at a time (default: parallel)")
	offline := flag.Bool("offline", false, "Skip web scraping, use curated picks only")
	workers := flag.Int("workers", 16, "Max concurrent HTTP requests across all agents")
	proxy := flag.String("proxy", "", "Proxy for all agents (default: $MOVIE_FINDER_PROXY or config proxy)")
	docket := flag.String("proxy-docket", "config/proxy-docket.json", "Proxy docket file (set to empty to disable)")
	testProxies := flag.Bool("test-proxies", false, "Test proxies in docket and exit")
	timeout := flag.Duration("timeout", 45*time.Second, "Hard max runtime before giving up on network calls")

	flag.Parse()

	return Options{
		Config:      *config,
		Sequential:  *sequential,
		Offline:     *offline,
		Workers:     *workers,
		Proxy:       *proxy,
		Docket:      *docket,
		TestProxies: *testProxies,
		Timeout:     *timeout,
	}
}

func (o Options) EnvProxy() string {
	return os.Getenv("MOVIE_FINDER_PROXY")
}
