package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/murops/forge/backend/internal/requests"
)

type fakeStore struct {
	req    requests.Request
	getErr error
	status requests.Status
	url    string
	errMsg string
}

func (s *fakeStore) Get(context.Context, uuid.UUID) (requests.Request, error) {
	return s.req, s.getErr
}

func (s *fakeStore) MarkRunning(context.Context, uuid.UUID) error {
	s.status = requests.StatusRunning
	return nil
}

func (s *fakeStore) MarkSucceeded(_ context.Context, _ uuid.UUID, url string) error {
	s.status, s.url = requests.StatusSucceeded, url
	return nil
}

func (s *fakeStore) MarkFailed(_ context.Context, _ uuid.UUID, msg string) error {
	s.status, s.errMsg = requests.StatusFailed, msg
	return nil
}

type fakeProvisioner struct {
	url   string
	err   error
	calls int
}

func (p *fakeProvisioner) Provision(context.Context, requests.Request) (string, error) {
	p.calls++
	return p.url, p.err
}

func newJob(attempt, maxAttempts int) *river.Job[requests.ProvisionArgs] {
	return &river.Job[requests.ProvisionArgs]{
		JobRow: &rivertype.JobRow{Attempt: attempt, MaxAttempts: maxAttempts},
		Args:   requests.ProvisionArgs{RequestID: uuid.New()},
	}
}

func newWorker(s Store, p Provisioner) *ProvisionWorker {
	return NewProvisionWorker(s, p, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestWorkSucceeds(t *testing.T) {
	s := &fakeStore{req: requests.Request{Status: requests.StatusPending}}
	p := &fakeProvisioner{url: "https://github.com/org/repo"}

	if err := newWorker(s, p).Work(context.Background(), newJob(1, 3)); err != nil {
		t.Fatal(err)
	}
	if s.status != requests.StatusSucceeded || s.url != p.url {
		t.Errorf("status = %s, url = %q", s.status, s.url)
	}
}

func TestWorkRetriesBeforeLastAttempt(t *testing.T) {
	s := &fakeStore{req: requests.Request{Status: requests.StatusPending}}
	p := &fakeProvisioner{err: errors.New("github is down")}

	if err := newWorker(s, p).Work(context.Background(), newJob(1, 3)); err == nil {
		t.Fatal("expected error so River retries the job")
	}
	if s.status != requests.StatusRunning {
		t.Errorf("status = %s, want running", s.status)
	}
}

func TestWorkFailsOnLastAttempt(t *testing.T) {
	s := &fakeStore{req: requests.Request{Status: requests.StatusRunning}}
	p := &fakeProvisioner{err: errors.New("github is down")}

	if err := newWorker(s, p).Work(context.Background(), newJob(3, 3)); err == nil {
		t.Fatal("expected error")
	}
	if s.status != requests.StatusFailed || s.errMsg != "github is down" {
		t.Errorf("status = %s, error = %q", s.status, s.errMsg)
	}
}

func TestWorkSkipsFinishedRequest(t *testing.T) {
	s := &fakeStore{req: requests.Request{Status: requests.StatusSucceeded}}
	p := &fakeProvisioner{}

	if err := newWorker(s, p).Work(context.Background(), newJob(1, 3)); err != nil {
		t.Fatal(err)
	}
	if p.calls != 0 {
		t.Errorf("provisioner called %d times", p.calls)
	}
}

func TestWorkCancelsWhenRequestIsGone(t *testing.T) {
	s := &fakeStore{getErr: requests.ErrNotFound}

	err := newWorker(s, &fakeProvisioner{}).Work(context.Background(), newJob(1, 3))
	var cancel *river.JobCancelError
	if !errors.As(err, &cancel) {
		t.Errorf("expected JobCancelError, got %v", err)
	}
}
