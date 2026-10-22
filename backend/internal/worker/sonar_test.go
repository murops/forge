package worker

import (
	"context"
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
