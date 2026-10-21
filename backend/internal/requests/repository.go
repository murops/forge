package requests

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/murops/forge/backend/internal/store"
)

type Repository struct {
	q *store.Queries
}

func NewRepository(db store.DBTX) *Repository {
	return &Repository{q: store.New(db)}
}

func (r *Repository) withTx(tx pgx.Tx) *Repository {
	return &Repository{q: r.q.WithTx(tx)}
}

func (r *Repository) Templates(ctx context.Context) ([]Template, error) {
	rows, err := r.q.ListTemplates(ctx)
	if err != nil {
		return nil, err
	}
	ts := make([]Template, len(rows))
	for i, row := range rows {
		ts[i] = toTemplate(row)
	}
	return ts, nil
}

func (r *Repository) Template(ctx context.Context, slug string) (Template, error) {
	row, err := r.q.GetTemplate(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return Template{}, ErrTemplateNotFound
	}
	return toTemplate(row), err
}

func (r *Repository) Create(ctx context.Context, in CreateInput, private bool, u User) (Request, error) {
	row, err := r.q.CreateRepoRequest(ctx, store.CreateRepoRequestParams{
		Name:             in.Name,
		Description:      in.Description,
		TemplateSlug:     in.Template,
		Private:          private,
		WithLambda:       in.WithLambda,
		RequestedBy:      u.Subject,
		RequestedByEmail: u.Email,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return Request{}, ErrNameTaken
	}
	return toRequest(row), err
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (Request, error) {
	row, err := r.q.GetRepoRequest(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, ErrNotFound
	}
	return toRequest(row), err
}

func (r *Repository) ListByUser(ctx context.Context, subject string) ([]Request, error) {
	rows, err := r.q.ListRepoRequestsByUser(ctx, subject)
	if err != nil {
		return nil, err
	}
	rs := make([]Request, len(rows))
	for i, row := range rows {
		rs[i] = toRequest(row)
	}
	return rs, nil
}

func (r *Repository) MarkRunning(ctx context.Context, id uuid.UUID) error {
	return r.q.MarkRepoRequestRunning(ctx, id)
}

func (r *Repository) MarkSucceeded(ctx context.Context, id uuid.UUID, repoURL string) error {
	return r.q.MarkRepoRequestSucceeded(ctx, store.MarkRepoRequestSucceededParams{ID: id, RepoUrl: &repoURL})
}

func (r *Repository) MarkFailed(ctx context.Context, id uuid.UUID, msg string) error {
	return r.q.MarkRepoRequestFailed(ctx, store.MarkRepoRequestFailedParams{ID: id, Error: &msg})
}

func toTemplate(t store.Template) Template {
	return Template{
		Slug:           t.Slug,
		Name:           t.Name,
		Description:    t.Description,
		SourceRepo:     t.SourceRepo,
		SupportsLambda: t.SupportsLambda,
	}
}

func toRequest(r store.RepoRequest) Request {
	return Request{
		ID:               r.ID,
		Name:             r.Name,
		Description:      r.Description,
		Template:         r.TemplateSlug,
		Private:          r.Private,
		WithLambda:       r.WithLambda,
		Status:           Status(r.Status),
		RepoURL:          r.RepoUrl,
		Error:            r.Error,
		RequestedBy:      r.RequestedBy,
		RequestedByEmail: r.RequestedByEmail,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}
