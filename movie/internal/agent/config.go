package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if len(cfg.Agents) == 0 {
		return Config{}, fmt.Errorf("config has no agents — add at least one to %s", path)
	}

	for i := range cfg.Agents {
		if cfg.Agents[i].Limit <= 0 {
			cfg.Agents[i].Limit = 3
		}
		if cfg.Agents[i].Discover == nil {
			on := true
			cfg.Agents[i].Discover = &on
		}
		if cfg.Agents[i].MaxSubreddits <= 0 {
			cfg.Agents[i].MaxSubreddits = 20
		}
		if cfg.Agents[i].MaxThreads <= 0 {
			cfg.Agents[i].MaxThreads = 25
		}
		if len(cfg.Agents[i].Subreddits) == 0 {
			cfg.Agents[i].Subreddits = []string{"MovieSuggestions", "movies"}
		}
	}
	ApplyEnv(&cfg)
	return cfg, nil
}

func ApplyEnv(cfg *Config) {
	if v := strings.TrimSpace(os.Getenv("MOVIE_PROXY")); v != "" {
		cfg.Proxy = v
	}
	for _, raw := range strings.Split(os.Getenv("MOVIE_PROXIES"), ",") {
		raw = strings.TrimSpace(raw)
		if raw != "" {
			cfg.Proxies = append(cfg.Proxies, raw)
		}
	}
}

func FilterByID(cfg Config, id string) (Config, error) {
	id = strings.TrimSpace(id)
	for _, a := range cfg.Agents {
		if a.ID == id {
			cfg.Agents = []Agent{a}
			return cfg, nil
		}
	}
	return Config{}, fmt.Errorf("unknown agent id %q", id)
}

func FilterByIndex(cfg Config, idx int) (Config, error) {
	if idx < 0 || idx >= len(cfg.Agents) {
		return Config{}, fmt.Errorf("agent index %d out of range (have %d agents)", idx, len(cfg.Agents))
	}
	cfg.Agents = []Agent{cfg.Agents[idx]}
	return cfg, nil
}
