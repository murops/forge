package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newSonarServer(t *testing.T, createStatus int, createBody string) (*SonarCloud, *[]string) {
	t.Helper()
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		calls = append(calls, r.URL.Path)
		switch r.URL.Path {
		case "/api/projects/create":
			if r.PostForm.Get("project") != "org_orders" || r.PostForm.Get("organization") != "org" {
				t.Errorf("unexpected form: %v", r.PostForm)
			}
			w.WriteHeader(createStatus)
			w.Write([]byte(createBody))
		case "/api/autoscan/activation":
			if r.PostForm.Get("enable") != "false" {
				t.Errorf("unexpected form: %v", r.PostForm)
			}
		case "/api/settings/set":
			if r.PostForm.Get("value") != "(main|homol|desen)" {
				t.Errorf("unexpected form: %v", r.PostForm)
			}
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(srv.Close)

	s := NewSonarCloud("token", "org")
	s.BaseURL = srv.URL
	return s, &calls
}

func TestEnsureProjectCreates(t *testing.T) {
	s, calls := newSonarServer(t, http.StatusOK, `{"project":{"key":"org_orders"}}`)

	if err := s.EnsureProject(context.Background(), "org_orders", "orders"); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 3 {
		t.Errorf("calls = %v", *calls)
	}
}

func TestEnsureProjectAlreadyExists(t *testing.T) {
	s, calls := newSonarServer(t, http.StatusBadRequest,
		`{"errors":[{"msg":"Could not create Project, key already exists: org_orders"}]}`)

	if err := s.EnsureProject(context.Background(), "org_orders", "orders"); err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 3 {
		t.Errorf("settings must still be applied, calls = %v", *calls)
	}
}

func TestEnsureProjectError(t *testing.T) {
	s, _ := newSonarServer(t, http.StatusForbidden, `{"errors":[{"msg":"Insufficient privileges"}]}`)

	err := s.EnsureProject(context.Background(), "org_orders", "orders")
	if err == nil || err.Error() != "/api/projects/create: 403 Insufficient privileges" {
		t.Errorf("err = %v", err)
	}
}

type fakeBinding struct {
	bound   string // slug the project is already bound to, "" when unbound
	project bool   // whether the project exists
	posted  map[string]string
	calls   []string
}

func newSonarAPIServer(t *testing.T, b *fakeBinding) *SonarCloud {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		b.calls = append(b.calls, r.Method+" "+r.URL.Path)
		q := r.URL.Query()
		switch r.Method + " " + r.URL.Path {
		case "GET /organizations/organizations":
			if q.Get("organizationKey") != "org" {
				t.Errorf("organizationKey = %q", q.Get("organizationKey"))
			}
			w.Write([]byte(`[{"key":"org","uuidV4":"org-uuid"}]`))
		case "GET /projects/projects":
			if q.Get("keys") != "org_orders" || q.Get("organizationIds") != "org-uuid" {
				t.Errorf("unexpected query: %v", q)
			}
			if b.project {
				w.Write([]byte(`{"projects":[{"key":"org_orders","legacyId":"AY123"}]}`))
			} else {
				w.Write([]byte(`{"projects":[]}`))
			}
		case "GET /dop-translation/project-bindings":
			if q.Get("projectId") != "AY123" {
				t.Errorf("projectId = %q", q.Get("projectId"))
			}
			if b.bound != "" {
				w.Write([]byte(`{"bindings":[{"slug":"` + b.bound + `"}]}`))
			} else {
				w.Write([]byte(`{"bindings":[]}`))
			}
		case "POST /dop-translation/project-bindings":
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("content type = %q", r.Header.Get("Content-Type"))
			}
			if err := json.NewDecoder(r.Body).Decode(&b.posted); err != nil {
				t.Fatal(err)
			}
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id":"binding-1"}`))
		case "POST /api/autoscan/activation":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.PostForm.Get("enable") != "false" || r.PostForm.Get("projectKey") != "org_orders" {
				t.Errorf("unexpected form: %v", r.PostForm)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message":"unknown path"}`))
		}
	}))
	t.Cleanup(srv.Close)

	s := NewSonarCloud("token", "org")
	s.BaseURL = srv.URL
	s.APIURL = srv.URL
	return s
}

func TestBindRepositoryBinds(t *testing.T) {
	b := &fakeBinding{project: true}
	s := newSonarAPIServer(t, b)

	if err := s.BindRepository(context.Background(), "org_orders", "org/orders"); err != nil {
		t.Fatal(err)
	}
	if b.posted["projectId"] != "AY123" || b.posted["repositoryId"] != "org/orders" {
		t.Errorf("posted = %v", b.posted)
	}
	if last := b.calls[len(b.calls)-1]; last != "POST /api/autoscan/activation" {
		t.Errorf("automatic analysis must be turned off after binding, calls = %v", b.calls)
	}
}

func TestBindRepositoryAlreadyBound(t *testing.T) {
	b := &fakeBinding{project: true, bound: "Org/Orders"}
	s := newSonarAPIServer(t, b)

	if err := s.BindRepository(context.Background(), "org_orders", "org/orders"); err != nil {
		t.Fatal(err)
	}
	if b.posted != nil {
		t.Errorf("must not bind again, posted = %v", b.posted)
	}
}

func TestBindRepositoryBoundElsewhere(t *testing.T) {
	b := &fakeBinding{project: true, bound: "org/other"}
	s := newSonarAPIServer(t, b)

	err := s.BindRepository(context.Background(), "org_orders", "org/orders")
	if err == nil || err.Error() != `project "org_orders" is already bound to org/other` {
		t.Errorf("err = %v", err)
	}
	if b.posted != nil {
		t.Errorf("must not rebind, posted = %v", b.posted)
	}
}

func TestBindRepositoryProjectMissing(t *testing.T) {
	s := newSonarAPIServer(t, &fakeBinding{})

	err := s.BindRepository(context.Background(), "org_orders", "org/orders")
	if err == nil || err.Error() != `project "org_orders" not found` {
		t.Errorf("err = %v", err)
	}
}

func TestBindRepositoryAPIError(t *testing.T) {
	s := NewSonarCloud("wrong", "org")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"Insufficient privileges"}`))
	}))
	t.Cleanup(srv.Close)
	s.APIURL = srv.URL

	err := s.BindRepository(context.Background(), "org_orders", "org/orders")
	if err == nil || err.Error() != "/organizations/organizations: 403 Insufficient privileges" {
		t.Errorf("err = %v", err)
	}
}
