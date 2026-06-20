package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func LoadDocket(path string) (Docket, error) {
	if strings.TrimSpace(path) == "" {
		return Docket{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Docket{}, fmt.Errorf("proxy docket not found: %s", path)
		}
		return Docket{}, fmt.Errorf("read proxy docket: %w", err)
	}

	var docket Docket
	if err := json.Unmarshal(data, &docket); err != nil {
		return Docket{}, fmt.Errorf("parse proxy docket: %w", err)
	}
	return docket, nil
}

func IsPlaceholderURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return true
	}
	return strings.Contains(raw, "YOUR-PROXY-HOST") ||
		strings.Contains(raw, "YOUR_VPS_IP") ||
		strings.Contains(raw, "USER:PASS@")
}

func EnabledProxies(docket Docket) []string {
	var urls []string
	for _, entry := range docket.Proxies {
		if !entry.Enabled {
			continue
		}
		raw := strings.TrimSpace(entry.URL)
		if IsPlaceholderURL(raw) {
			continue
		}
		urls = append(urls, raw)
	}
	return urls
}

func PickFromDocket(docket Docket, agentIdx int) string {
	if id := strings.TrimSpace(docket.DefaultProxy); id != "" {
		for _, entry := range docket.Proxies {
			if entry.ID != id || !entry.Enabled {
				continue
			}
			raw := strings.TrimSpace(entry.URL)
			if !IsPlaceholderURL(raw) {
				return raw
			}
		}
	}

	proxies := EnabledProxies(docket)
	if len(proxies) == 0 {
		return ""
	}
	return proxies[agentIdx%len(proxies)]
}

func testEntry(ctx context.Context, pool *ClientPool, entry Entry) error {
	raw := strings.TrimSpace(entry.URL)
	client := pool.Get(raw)

	resp, err := client.Get(ctx, "movie-finder-proxy-test/1.0", "https://www.reddit.com/.json")
	if err != nil {
		return fmt.Errorf("%s (%s): request failed: %w", entry.ID, Mask(raw), err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	switch resp.StatusCode {
	case http.StatusOK, http.StatusForbidden, http.StatusTooManyRequests:
		fmt.Printf("  OK  %s — %s (reddit responded %d)\n", entry.ID, Mask(raw), resp.StatusCode)
		return nil
	default:
		return fmt.Errorf("%s (%s): unexpected status %d", entry.ID, Mask(raw), resp.StatusCode)
	}
}

func TestProxies(ctx context.Context, pool *ClientPool, docket Docket, flagProxy, envProxy string) error {
	var entries []Entry

	if raw := strings.TrimSpace(flagProxy); raw != "" {
		if err := ValidateURL(raw); err != nil {
			return fmt.Errorf("-proxy: %w", err)
		}
		entries = append(entries, Entry{ID: "cli", Name: "CLI -proxy", URL: raw})
	} else if raw := strings.TrimSpace(envProxy); raw != "" {
		if err := ValidateURL(raw); err != nil {
			return fmt.Errorf("MOVIE_FINDER_PROXY: %w", err)
		}
		entries = append(entries, Entry{ID: "env", Name: "MOVIE_FINDER_PROXY", URL: raw})
	}

	for _, entry := range docket.Proxies {
		if !entry.Enabled {
			continue
		}
		raw := strings.TrimSpace(entry.URL)
		if raw == "" {
			return fmt.Errorf("proxy %q is enabled but has an empty url", entry.ID)
		}
		if IsPlaceholderURL(raw) {
			return fmt.Errorf("proxy %q still has placeholder values — edit proxy-docket.json", entry.ID)
		}
		if err := ValidateURL(raw); err != nil {
			return fmt.Errorf("proxy %q: %w", entry.ID, err)
		}
		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return fmt.Errorf("no proxies to test — pass -proxy, set MOVIE_FINDER_PROXY, or enable entries in proxy-docket.json (see config/PROXY-SETUP.md)")
	}

	fmt.Println("Testing proxies...")
	var failures []string
	for _, entry := range entries {
		if err := testEntry(ctx, pool, entry); err != nil {
			fmt.Printf("  FAIL %s — %v\n", entry.ID, err)
			failures = append(failures, entry.ID)
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%d proxy test(s) failed: %s", len(failures), strings.Join(failures, ", "))
	}
	fmt.Println("All proxies passed basic connectivity test.")
	return nil
}
