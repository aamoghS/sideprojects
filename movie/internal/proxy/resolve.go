package proxy

import (
	"os"
	"strings"

	"movie/internal/agent"
)

func Resolve(a agent.Agent, agentIdx int, cfg agent.Config, docket Docket, flagProxy string) (string, error) {
	candidates := []string{
		a.Proxy,
		PickFromDocket(docket, agentIdx),
		pickFromPool(cfg.Proxies, agentIdx),
		cfg.Proxy,
		flagProxy,
		os.Getenv("MOVIE_FINDER_PROXY"),
		os.Getenv("HTTPS_PROXY"),
		os.Getenv("HTTP_PROXY"),
	}

	for _, raw := range candidates {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if err := ValidateURL(raw); err != nil {
			return "", err
		}
		return raw, nil
	}
	return "", nil
}

func pickFromPool(proxies []string, agentIdx int) string {
	if len(proxies) == 0 {
		return ""
	}
	return strings.TrimSpace(proxies[agentIdx%len(proxies)])
}

func FirstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
