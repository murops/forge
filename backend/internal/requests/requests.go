package requests

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
)

type Template struct {
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	SourceRepo     string `json:"source_repo"`
	SupportsLambda bool   `json:"supports_lambda"`
}

type Request struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Template         string    `json:"template"`
	Private          bool      `json:"private"`
	WithLambda       bool      `json:"with_lambda"`
	Status           Status    `json:"status"`
	RepoURL          *string   `json:"repo_url"`
	Error            *string   `json:"error"`
	RequestedBy      string    `json:"requested_by"`
	RequestedByEmail string    `json:"requested_by_email"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type User struct {
	Subject string
	Email   string
}

type CreateInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Template    string `json:"template"`
	Private     *bool  `json:"private"`
	WithLambda  bool   `json:"with_lambda"`
}

var (
	ErrNotFound         = errors.New("not found")
	ErrTemplateNotFound = errors.New("template not found")
	ErrNameTaken        = errors.New("a repository with this name already exists or is being created")
)

type ValidationError struct {
	Field string
	Msg   string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Msg)
}

var nameRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

func (in CreateInput) Validate() error {
	switch {
	case len(in.Name) < 3 || len(in.Name) > 100:
		return &ValidationError{"name", "must be between 3 and 100 characters"}
	case !nameRe.MatchString(in.Name):
		return &ValidationError{"name", "only lowercase letters, digits and hyphens, not starting or ending with a hyphen"}
	case len(in.Description) > 350:
		return &ValidationError{"description", "must be at most 350 characters"}
	case in.Template == "":
		return &ValidationError{"template", "is required"}
	}
	return nil
}
