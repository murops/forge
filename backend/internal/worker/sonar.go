package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SonarCloud struct {
	BaseURL string
	// APIURL serves Web API v2, which the repository binding needs.
	APIURL       string
	Token        string
	Organization string
	HTTP         *http.Client
}

func NewSonarCloud(token, organization string) *SonarCloud {
	return &SonarCloud{
		BaseURL:      "https://sonarcloud.io",
		APIURL:       "https://api.sonarcloud.io",
		Token:        token,
		Organization: organization,
		HTTP:         &http.Client{Timeout: 30 * time.Second},
	}
}

// EnsureProject creates the project if needed, turns off automatic analysis
// (it rejects analyses coming from CI) and marks the environment branches as
// long-lived.
func (s *SonarCloud) EnsureProject(ctx context.Context, key, name string) error {
	err := s.post(ctx, "/api/projects/create", url.Values{
		"organization": {s.Organization},
		"project":      {key},
		"name":         {name},
		"visibility":   {"public"},
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}
	if err := s.disableAutoscan(ctx, key); err != nil {
		return err
	}
	return s.post(ctx, "/api/settings/set", url.Values{
		"component": {key},
		"key":       {"sonar.branch.longLivedBranches.regex"},
		"value":     {"(main|homol|desen)"},
	})
}

func (s *SonarCloud) disableAutoscan(ctx context.Context, key string) error {
	return s.post(ctx, "/api/autoscan/activation", url.Values{
		"projectKey": {key},
		"enable":     {"false"},
	})
}

// BindRepository binds the project to its GitHub repository ("owner/name").
// Without a binding, importing the repository from SonarCloud's UI creates a
// second project ("<key>2") with automatic analysis, and every pull request
// gets two analyses. The repository must already exist.
//
// The binding endpoints are internal to Web API v2; the calls follow
// SonarSource's own Terraform provider. They name the project by its legacy
// id and take the GitHub repository as its slug.
func (s *SonarCloud) BindRepository(ctx context.Context, key, repository string) error {
	orgID, err := s.organizationID(ctx)
	if err != nil {
		return err
	}
	projectID, err := s.projectID(ctx, orgID, key)
	if err != nil {
		return err
	}
	slug, err := s.boundRepository(ctx, projectID)
	if err != nil {
		return err
	}
	switch {
	case slug == "":
		body := map[string]string{"projectId": projectID, "repositoryId": repository}
		if err := s.apiSend(ctx, http.MethodPost, "/dop-translation/project-bindings", nil, body, nil); err != nil {
			return err
		}
	case !strings.EqualFold(slug, repository):
		return fmt.Errorf("project %q is already bound to %s", key, slug)
	}
	// a bound project may come back with automatic analysis on, which rejects CI analyses
	return s.disableAutoscan(ctx, key)
}

// organizationID returns the UUID that Web API v2 uses for the organization.
func (s *SonarCloud) organizationID(ctx context.Context) (string, error) {
	var orgs []struct {
		Key    string `json:"key"`
		UUIDV4 string `json:"uuidV4"`
	}
	if err := s.apiGet(ctx, "/organizations/organizations", url.Values{"organizationKey": {s.Organization}}, &orgs); err != nil {
		return "", err
	}
	for _, o := range orgs {
		if o.Key == s.Organization {
			return o.UUIDV4, nil
		}
	}
	return "", fmt.Errorf("organization %q not found", s.Organization)
}

// projectID returns the legacy id, the only project id the bindings API takes.
func (s *SonarCloud) projectID(ctx context.Context, orgID, key string) (string, error) {
	var out struct {
		Projects []struct {
			Key      string `json:"key"`
			LegacyID string `json:"legacyId"`
		} `json:"projects"`
	}
	if err := s.apiGet(ctx, "/projects/projects", url.Values{"keys": {key}, "organizationIds": {orgID}}, &out); err != nil {
		return "", err
	}
	for _, p := range out.Projects {
		if p.Key == key {
			return p.LegacyID, nil
		}
	}
	return "", fmt.Errorf("project %q not found", key)
}

// boundRepository returns the "owner/name" the project is bound to, or "".
func (s *SonarCloud) boundRepository(ctx context.Context, projectID string) (string, error) {
	var out struct {
		Bindings []struct {
			Slug string `json:"slug"`
		} `json:"bindings"`
	}
	if err := s.apiGet(ctx, "/dop-translation/project-bindings", url.Values{"projectId": {projectID}}, &out); err != nil {
		return "", err
	}
	if len(out.Bindings) == 0 {
		return "", nil
	}
	return out.Bindings[0].Slug, nil
}

// projectKeys lists every project key of the organization (Web API v1).
func (s *SonarCloud) projectKeys(ctx context.Context) ([]string, error) {
	var keys []string
	for page := 1; ; page++ {
		var out struct {
			Components []struct {
				Key string `json:"key"`
			} `json:"components"`
			Paging struct {
				Total int `json:"total"`
			} `json:"paging"`
		}
		query := url.Values{"organization": {s.Organization}, "ps": {"500"}, "p": {fmt.Sprint(page)}}
		if err := s.get(ctx, "/api/projects/search", query, &out); err != nil {
			return nil, err
		}
		for _, c := range out.Components {
			keys = append(keys, c.Key)
		}
		if len(out.Components) == 0 || len(keys) >= out.Paging.Total {
			return keys, nil
		}
	}
}

func (s *SonarCloud) apiGet(ctx context.Context, path string, query url.Values, out any) error {
	return s.apiSend(ctx, http.MethodGet, path, query, nil, out)
}

func (s *SonarCloud) apiSend(ctx context.Context, method, path string, query url.Values, body, out any) error {
	u := s.APIURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return apiError(path, resp)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func (s *SonarCloud) get(ctx context.Context, path string, query url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.BaseURL+path+"?"+query.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	resp, err := s.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return apiError(path, resp)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (s *SonarCloud) post(ctx context.Context, path string, form url.Values) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.BaseURL+path, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 300 {
		return nil
	}

	return apiError(path, resp)
}

// apiError reads the message of both API versions: v1 answers
// {"errors":[{"msg":...}]}, v2 answers {"message":...}.
func apiError(path string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var apiErr struct {
		Errors []struct {
			Msg string `json:"msg"`
		} `json:"errors"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &apiErr) == nil {
		if len(apiErr.Errors) > 0 {
			return fmt.Errorf("%s: %d %s", path, resp.StatusCode, apiErr.Errors[0].Msg)
		}
		if apiErr.Message != "" {
			return fmt.Errorf("%s: %d %s", path, resp.StatusCode, apiErr.Message)
		}
	}
	return fmt.Errorf("%s: %d", path, resp.StatusCode)
}
