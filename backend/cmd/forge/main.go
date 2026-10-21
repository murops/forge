package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/murops/forge/backend/internal/config"
	"github.com/murops/forge/backend/internal/db"
	"github.com/murops/forge/backend/internal/httpapi"
	"github.com/murops/forge/backend/internal/requests"
	"github.com/murops/forge/backend/internal/worker"
)

const usage = "usage: forge <api|worker|migrate>"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch os.Args[1] {
	case "api":
		err = runAPI(ctx, cfg, log)
	case "worker":
		err = runWorker(ctx, cfg, log)
	case "migrate":
		err = runMigrate(ctx, cfg, log)
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		log.Error("exiting", "error", err)
		os.Exit(1)
	}
}

func runMigrate(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	return db.Migrate(ctx, pool, log)
}

func runAPI(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	// insert-only client, jobs are worked by the worker process
	queue, err := river.NewClient(riverpgxv5.New(pool), &river.Config{Logger: log})
	if err != nil {
		return err
	}

	verifier, err := httpapi.NewOIDCVerifier(ctx, cfg.OIDCIssuer, cfg.OIDCDiscoveryURL, cfg.OIDCAudience)
	if err != nil {
		return fmt.Errorf("oidc: %w", err)
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           httpapi.NewRouter(requests.NewService(pool, queue), verifier, pool, log),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		log.Info("api listening", "addr", cfg.HTTPAddr)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func runWorker(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	provisioner, err := newProvisioner(cfg, log)
	if err != nil {
		return err
	}
	log.Info("using provisioner", "type", cfg.Provisioner)

	workers := river.NewWorkers()
	river.AddWorker(workers, worker.NewProvisionWorker(requests.NewRepository(pool), provisioner, log))

	client, err := river.NewClient[pgx.Tx](riverpgxv5.New(pool), &river.Config{
		Logger:  log,
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: cfg.WorkerConcurrency}},
		Workers: workers,
	})
	if err != nil {
		return err
	}
	if err := client.Start(ctx); err != nil {
		return err
	}
	log.Info("worker started", "concurrency", cfg.WorkerConcurrency)

	<-ctx.Done()
	log.Info("shutting down, waiting for running jobs")
	stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return client.Stop(stopCtx)
}

func newProvisioner(cfg config.Config, log *slog.Logger) (worker.Provisioner, error) {
	if cfg.Provisioner == "dryrun" {
		return worker.DryRun{Org: cfg.GitHubOrg, Delay: 3 * time.Second, Log: log}, nil
	}

	app, err := worker.LoadGitHubApp(cfg.GitHubAppID, cfg.GitHubAppInstallationID, cfg.GitHubAppPrivateKeyFile)
	if err != nil {
		return nil, err
	}
	bin, err := exec.LookPath(cfg.TerraformBin)
	if err != nil {
		return nil, fmt.Errorf("terraform binary: %w", err)
	}
	return &worker.Terraform{
		ExecPath:        bin,
		InfraDir:        cfg.TerraformDir,
		StateConnString: cfg.DatabaseURL,
		Owner:           cfg.GitHubOrg,
		App:             app,
		PrivateRulesets: cfg.GitHubPrivateRulesets,
		AWS: worker.AWSOptions{
			Enabled:             cfg.AWSEnabled,
			Region:              cfg.AWSRegion,
			OIDCProviderARN:     cfg.AWSGitHubOIDCProviderARN,
			PlaceholderImageURI: cfg.LambdaPlaceholderImageURI,
		},
		Log: log,
	}, nil
}
