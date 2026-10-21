package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/murops/forge/backend/internal/requests"
)

type fakeVerifier struct{}

func (fakeVerifier) Verify(_ context.Context, raw string) (requests.User, error) {
	if raw != "good" {
		return requests.User{}, errors.New("bad token")
	}
	return requests.User{Subject: "user-1", Email: "dev@example.com"}, nil
}

type fakePinger struct{ err error }

func (p fakePinger) Ping(context.Context) error { return p.err }

type fakeService struct {
	created []requests.CreateInput
	reqs    map[uuid.UUID]requests.Request
}

func (s *fakeService) Templates(context.Context) ([]requests.Template, error) {
	return []requests.Template{{Slug: "api-go", Name: "API Go"}}, nil
}

func (s *fakeService) Create(_ context.Context, in requests.CreateInput, u requests.User) (requests.Request, error) {
	if err := in.Validate(); err != nil {
		return requests.Request{}, err
	}
	if in.Template != "api-go" {
		return requests.Request{}, requests.ErrTemplateNotFound
	}
	if in.Name == "taken" {
		return requests.Request{}, requests.ErrNameTaken
	}
	s.created = append(s.created, in)
	return requests.Request{ID: uuid.New(), Name: in.Name, Status: requests.StatusPending, RequestedBy: u.Subject}, nil
}

func (s *fakeService) Get(_ context.Context, id uuid.UUID, u requests.User) (requests.Request, error) {
	r, ok := s.reqs[id]
	if !ok || r.RequestedBy != u.Subject {
		return requests.Request{}, requests.ErrNotFound
	}
	return r, nil
}

func (s *fakeService) List(context.Context, requests.User) ([]requests.Request, error) {
	return nil, nil
}

func newTestServer(svc *fakeService, db Pinger) http.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(svc, fakeVerifier{}, db, log)
}

func do(t *testing.T, h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAuthRequired(t *testing.T) {
	h := newTestServer(&fakeService{}, fakePinger{})

	if rec := do(t, h, "GET", "/api/v1/templates", "", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: got %d", rec.Code)
	}
	if rec := do(t, h, "GET", "/api/v1/templates", "bad", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("bad token: got %d", rec.Code)
	}
	if rec := do(t, h, "GET", "/api/v1/templates", "good", ""); rec.Code != http.StatusOK {
		t.Errorf("good token: got %d", rec.Code)
	}
}

func TestCreateRepository(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"accepted", `{"name":"payments-api","template":"api-go"}`, http.StatusAccepted},
		{"invalid json", `{"name":`, http.StatusBadRequest},
		{"invalid name", `{"name":"Payments","template":"api-go"}`, http.StatusUnprocessableEntity},
		{"unknown template", `{"name":"payments","template":"cobol"}`, http.StatusUnprocessableEntity},
		{"name taken", `{"name":"taken","template":"api-go"}`, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestServer(&fakeService{}, fakePinger{})
			rec := do(t, h, "POST", "/api/v1/repositories", "good", tt.body)
			if rec.Code != tt.want {
				t.Fatalf("got %d, want %d: %s", rec.Code, tt.want, rec.Body)
			}
		})
	}
}

func TestCreateRepositoryResponse(t *testing.T) {
	svc := &fakeService{}
	h := newTestServer(svc, fakePinger{})

	rec := do(t, h, "POST", "/api/v1/repositories", "good", `{"name":"payments-api","template":"api-go"}`)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("got %d", rec.Code)
	}

	var got requests.Request
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != requests.StatusPending || got.RequestedBy != "user-1" {
		t.Errorf("unexpected body: %+v", got)
	}
	if loc := rec.Header().Get("Location"); loc != "/api/v1/repositories/"+got.ID.String() {
		t.Errorf("Location = %q", loc)
	}
}

func TestGetRepository(t *testing.T) {
	mine, theirs := uuid.New(), uuid.New()
	svc := &fakeService{reqs: map[uuid.UUID]requests.Request{
		mine:   {ID: mine, RequestedBy: "user-1"},
		theirs: {ID: theirs, RequestedBy: "user-2"},
	}}
	h := newTestServer(svc, fakePinger{})

	cases := map[string]int{
		mine.String():   http.StatusOK,
		theirs.String(): http.StatusNotFound,
		"not-a-uuid":    http.StatusNotFound,
	}
	for id, want := range cases {
		if rec := do(t, h, "GET", "/api/v1/repositories/"+id, "good", ""); rec.Code != want {
			t.Errorf("%s: got %d, want %d", id, rec.Code, want)
		}
	}
}

func TestReadyz(t *testing.T) {
	if rec := do(t, newTestServer(&fakeService{}, fakePinger{}), "GET", "/readyz", "", ""); rec.Code != http.StatusOK {
		t.Errorf("healthy db: got %d", rec.Code)
	}
	down := fakePinger{err: errors.New("connection refused")}
	if rec := do(t, newTestServer(&fakeService{}, down), "GET", "/readyz", "", ""); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("db down: got %d", rec.Code)
	}
}
