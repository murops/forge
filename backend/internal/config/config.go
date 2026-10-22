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

	OIDCIssuer   string
	OIDCAudience string

	GitHubOrg         string
	WorkerConcurrency int

	Provisioner               string
	TerraformBin              string
	TerraformDir              string
	GitHubAppID               string
	GitHubAppInstallationID   string
	GitHubAppPrivateKeyFile   string
	GitHubPrivateRulesets     bool
	AWSEnabled                bool
	AWSRegion                 string
	AWSGitHubOIDCProviderARN  string
	LambdaPlaceholderImageURI string
	SonarToken                string
	SonarOrganization         string
}

func Load() (Config, error) {
	c := Config{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		HTTPAddr:     env("HTTP_ADDR", ":8000"),
		OIDCIssuer:   os.Getenv("OIDC_ISSUER"),
		OIDCAudience: env("OIDC_AUDIENCE", "forge-api"),
		GitHubOrg:    env("GITHUB_ORG", "murops-forge"),

		Provisioner:               env("PROVISIONER", "dryrun"),
		TerraformBin:              env("TERRAFORM_BIN", "terraform"),
		TerraformDir:              env("TERRAFORM_DIR", "../infra"),
		GitHubAppID:               os.Getenv("GITHUB_APP_ID"),
		GitHubAppInstallationID:   os.Getenv("GITHUB_APP_INSTALLATION_ID"),
		GitHubAppPrivateKeyFile:   os.Getenv("GITHUB_APP_PRIVATE_KEY_FILE"),
		GitHubPrivateRulesets:     os.Getenv("GITHUB_PRIVATE_RULESETS") == "true",
		AWSEnabled:                os.Getenv("AWS_ENABLED") == "true",
		AWSRegion:                 env("AWS_REGION", "us-east-1"),
		AWSGitHubOIDCProviderARN:  os.Getenv("AWS_GITHUB_OIDC_PROVIDER_ARN"),
		LambdaPlaceholderImageURI: os.Getenv("LAMBDA_PLACEHOLDER_IMAGE_URI"),
		SonarToken:                os.Getenv("SONAR_TOKEN"),
		SonarOrganization:         os.Getenv("SONAR_ORGANIZATION"),
	}
	if c.Provisioner != "dryrun" && c.Provisioner != "terraform" {
		return c, fmt.Errorf("PROVISIONER must be dryrun or terraform")
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
