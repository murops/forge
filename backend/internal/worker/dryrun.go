package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/murops/forge/backend/internal/requests"
)

// DryRun is used for local runs without GitHub credentials.
type DryRun struct {
	Org   string
	Delay time.Duration
	Log   *slog.Logger
}

func (d DryRun) Provision(ctx context.Context, req requests.Request) (string, error) {
	d.Log.Info("dry run: would create repository",
		"org", d.Org, "name", req.Name, "template", req.Template,
		"private", req.Private, "with_lambda", req.WithLambda)

	select {
	case <-time.After(d.Delay):
	case <-ctx.Done():
		return "", ctx.Err()
	}
	return fmt.Sprintf("https://github.com/%s/%s", d.Org, req.Name), nil
}
