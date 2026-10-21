package requests

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type ProvisionArgs struct {
	RequestID uuid.UUID `json:"request_id"`
}

func (ProvisionArgs) Kind() string { return "provision_repository" }

func (ProvisionArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{MaxAttempts: 3}
}

type Service struct {
	pool  *pgxpool.Pool
	repo  *Repository
	queue *river.Client[pgx.Tx]
}

func NewService(pool *pgxpool.Pool, queue *river.Client[pgx.Tx]) *Service {
	return &Service{pool: pool, repo: NewRepository(pool), queue: queue}
}

func (s *Service) Templates(ctx context.Context) ([]Template, error) {
	return s.repo.Templates(ctx)
}

// request row and job are inserted in the same transaction
func (s *Service) Create(ctx context.Context, in CreateInput, u User) (Request, error) {
	if err := in.Validate(); err != nil {
		return Request{}, err
	}
	private := false
	if in.Private != nil {
		private = *in.Private
	}

	var req Request
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		repo := s.repo.withTx(tx)
		tmpl, err := repo.Template(ctx, in.Template)
		if err != nil {
			return err
		}
		if in.WithLambda && !tmpl.SupportsLambda {
			return &ValidationError{"with_lambda", fmt.Sprintf("template %q does not support Lambda", tmpl.Slug)}
		}

		req, err = repo.Create(ctx, in, private, u)
		if err != nil {
			return err
		}
		_, err = s.queue.InsertTx(ctx, tx, ProvisionArgs{RequestID: req.ID}, nil)
		return err
	})
	return req, err
}

func (s *Service) Get(ctx context.Context, id uuid.UUID, u User) (Request, error) {
	req, err := s.repo.Get(ctx, id)
	if err != nil {
		return req, err
	}
	if req.RequestedBy != u.Subject {
		return Request{}, ErrNotFound
	}
	return req, nil
}

func (s *Service) List(ctx context.Context, u User) ([]Request, error) {
	return s.repo.ListByUser(ctx, u.Subject)
}
