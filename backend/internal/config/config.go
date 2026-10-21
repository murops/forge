package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL string
	HTTPAddr    string
	LogLevel    slog.Level

	OIDCIssuer       string
	OIDCDiscoveryURL string
	OIDCAudience     string

	GitHubOrg         string
	WorkerConcurrency int
}

func Load() (Config, error) {
	c := Config{
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		HTTPAddr:         env("HTTP_ADDR", ":8000"),
		OIDCIssuer:       os.Getenv("OIDC_ISSUER"),
		OIDCDiscoveryURL: os.Getenv("OIDC_DISCOVERY_URL"),
		OIDCAudience:     env("OIDC_AUDIENCE", "forge-api"),
		GitHubOrg:        env("GITHUB_ORG", "murops-forge"),
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	if err := c.LogLevel.UnmarshalText([]byte(env("LOG_LEVEL", "info"))); err != nil {
		return c, fmt.Errorf("LOG_LEVEL: %w", err)
	}

	n, err := strconv.Atoi(env("WORKER_CONCURRENCY", "5"))
	if err != nil || n < 1 {
		return c, fmt.Errorf("WORKER_CONCURRENCY must be a positive integer")
	}
	c.WorkerConcurrency = n

	return c, nil
}

func env(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
