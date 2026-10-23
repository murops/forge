package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeOrg is a SonarCloud organization: project key -> repository it is bound to ("" = unbound).
type fakeOrg struct {
	projects map[string]string
	posted   []map[string]string
}

func newSonarOrgServer(t *testing.T, o *fakeOrg) *SonarCloud {
	t.Helper()
	legacy := func(key string) string { return "L-" + key }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch r.Method + " " + r.URL.Path {
		case "GET /api/projects/search":
			var comps []map[string]string
			for k := range o.projects {
				comps = append(comps, map[string]string{"key": k})
			}
			json.NewEncoder(w).Encode(map[string]any{"components": comps, "paging": map[string]int{"total": len(comps)}})
		case "GET /organizations/organizations":
			w.Write([]byte(`[{"key":"org","uuidV4":"org-uuid"}]`))
		case "GET /projects/projects":
			key := q.Get("keys")
			if _, ok := o.projects[key]; !ok {
				w.Write([]byte(`{"projects":[]}`))
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"projects": []map[string]string{{"key": key, "legacyId": legacy(key)}}})
		case "GET /dop-translation/project-bindings":
			slug := ""
			for k, repo := range o.projects {
				if legacy(k) == q.Get("projectId") {
					slug = repo
				}
			}
			if slug == "" {
				w.Write([]byte(`{"bindings":[]}`))
				return
			}
			json.NewEncoder(w).Encode(map[string]any{"bindings": []map[string]string{{"slug": slug}}})
		case "POST /dop-translation/project-bindings":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			o.posted = append(o.posted, body)
			w.WriteHeader(http.StatusCreated)
		case "POST /api/autoscan/activation":
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	s := NewSonarCloud("token", "org")
	s.BaseURL, s.APIURL = srv.URL, srv.URL
	return s
}

func statuses(reports []BindReport) map[string]string {
	out := map[string]string{}
	for _, r := range reports {
		out[r.Project] = r.Status
	}
	return out
}

func TestBindExisting(t *testing.T) {
	o := &fakeOrg{projects: map[string]string{
		"own_api":          "",           // unbound: to bind
		"own_lambda":       "",           // unbound but has a duplicate: blocked
		"own_lambda2":      "own/lambda", // the duplicate SonarCloud's UI created
		"own_go":           "own/go",     // already bound
		"own_demo-2":       "",           // a repository whose name ends in 2, not a duplicate
		"someone-else_lib": "",           // not created by Forge
	}}
	s := newSonarOrgServer(t, o)

	dry, err := s.BindExisting(context.Background(), "own", false)
	if err != nil {
		t.Fatal(err)
	}
	got := statuses(dry)
	want := map[string]string{
		"own_api":          "would bind",
		"own_lambda":       "blocked: delete own_lambda2 first",
		"own_lambda2":      "duplicate of own_lambda: delete it in SonarCloud",
		"own_go":           "already bound",
		"own_demo-2":       "would bind",
		"someone-else_lib": "ignored: key does not follow <owner>_<repo>",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: status = %q, want %q", k, got[k], v)
		}
	}
	if len(o.posted) != 0 {
		t.Fatalf("a dry run must not bind, posted = %v", o.posted)
	}

	applied, err := s.BindExisting(context.Background(), "own", true)
	if err != nil {
		t.Fatal(err)
	}
	if st := statuses(applied); st["own_api"] != "bound" || st["own_demo-2"] != "bound" {
		t.Errorf("statuses = %v", st)
	}
	var repos []string
	for _, p := range o.posted {
		repos = append(repos, p["repositoryId"])
	}
	if strings.Join(repos, ",") != "own/api,own/demo-2" {
		t.Errorf("bound repositories = %v", repos)
	}
}
