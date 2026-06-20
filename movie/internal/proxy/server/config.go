package server

import (
	"fmt"
	"os"
	"strings"
)

// Config holds listen address and optional basic auth for the forward proxy.
type Config struct {
	Addr          string
	User          string
	Pass          string
	AuthRequired  bool
	AllowInsecure bool
}

// ConfigFromEnv loads production settings from PROXY_* environment variables.
func ConfigFromEnv() (Config, error) {
	cfg := Config{
		Addr:          envOr("PROXY_ADDR", "0.0.0.0:8888"),
		User:          strings.TrimSpace(os.Getenv("PROXY_USER")),
		Pass:          strings.TrimSpace(os.Getenv("PROXY_PASS")),
		AllowInsecure: envTruthy("ALLOW_INSECURE"),
	}
	return cfg, cfg.Validate()
}

// LocalConfig returns dev-friendly defaults (127.0.0.1, auth optional).
func LocalConfig(addr, user, pass string) (Config, error) {
	cfg := Config{
		Addr:          addr,
		User:          strings.TrimSpace(user),
		Pass:          strings.TrimSpace(pass),
		AllowInsecure: true,
	}
	return cfg, cfg.Validate()
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Addr) == "" {
		return fmt.Errorf("listen address is required")
	}

	authPartial := (c.User != "") != (c.Pass != "")
	if authPartial {
		return fmt.Errorf("basic auth requires both username and password")
	}

	c.AuthRequired = c.User != "" && c.Pass != ""
	if !c.AuthRequired && !c.AllowInsecure {
		return fmt.Errorf("PROXY_USER and PROXY_PASS are required in production (set ALLOW_INSECURE=true only for local dev)")
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envTruthy(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
