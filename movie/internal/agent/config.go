package agent

import (
	"encoding/json"
	"fmt"
	"os"
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
		if len(cfg.Agents[i].Subreddits) == 0 {
			cfg.Agents[i].Subreddits = []string{"MovieSuggestions", "movies"}
		}
	}
	return cfg, nil
}
