package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/murops/forge/backend/internal/requests"
)

type Provisioner interface {
	Provision(ctx context.Context, req requests.Request) (string, error)
}

type Store interface {
	Get(ctx context.Context, id uuid.UUID) (requests.Request, error)
	MarkRunning(ctx context.Context, id uuid.UUID) error
	MarkSucceeded(ctx context.Context, id uuid.UUID, repoURL string) error
	MarkFailed(ctx context.Context, id uuid.UUID, msg string) error
}

type ProvisionWorker struct {
	river.WorkerDefaults[requests.ProvisionArgs]
	store       Store
	provisioner Provisioner
	log         *slog.Logger
}

func NewProvisionWorker(s Store, p Provisioner, log *slog.Logger) *ProvisionWorker {
	return &ProvisionWorker{store: s, provisioner: p, log: log}
}

func (w *ProvisionWorker) Timeout(*river.Job[requests.ProvisionArgs]) time.Duration {
	return 10 * time.Minute
}

func (w *ProvisionWorker) Work(ctx context.Context, job *river.Job[requests.ProvisionArgs]) error {
	id := job.Args.RequestID
	log := w.log.With("request_id", id, "attempt", job.Attempt)

	req, err := w.store.Get(ctx, id)
	if errors.Is(err, requests.ErrNotFound) {
		return river.JobCancel(fmt.Errorf("request %s not found", id))
	}
	if err != nil {
		return err
	}
	if req.Status == requests.StatusSucceeded || req.Status == requests.StatusFailed {
		log.Info("request already finished, skipping", "status", req.Status)
		return nil
	}

	if err := w.store.MarkRunning(ctx, id); err != nil {
		return err
	}
	log.Info("provisioning repository", "name", req.Name, "template", req.Template)

	url, err := w.provisioner.Provision(ctx, req)
	if err != nil {
		log.Error("provisioning failed", "error", err)
		if job.Attempt >= job.MaxAttempts {
			// use a fresh context, the job's may already be cancelled
			if ferr := w.store.MarkFailed(context.WithoutCancel(ctx), id, err.Error()); ferr != nil {
				log.Error("could not mark request as failed", "error", ferr)
			}
		}
		return err
	}

	log.Info("repository ready", "url", url)
	return w.store.MarkSucceeded(ctx, id, url)
}
