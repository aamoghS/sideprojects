package seo

import (
	"net/http"
	"time"
)

func HTTPClientFromEnv() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			ResponseHeaderTimeout: 15 * time.Second,
		},
	}
}
