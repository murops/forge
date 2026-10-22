package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SonarCloud struct {
	BaseURL      string
	Token        string
	Organization string
	HTTP         *http.Client
}

func NewSonarCloud(token, organization string) *SonarCloud {
	return &SonarCloud{
		BaseURL:      "https://sonarcloud.io",
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
	err = s.post(ctx, "/api/autoscan/activation", url.Values{
		"projectKey": {key},
		"enable":     {"false"},
	})
	if err != nil {
		return err
	}
	return s.post(ctx, "/api/settings/set", url.Values{
		"component": {key},
		"key":       {"sonar.branch.longLivedBranches.regex"},
		"value":     {"(main|homol|desen)"},
	})
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

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var apiErr struct {
		Errors []struct {
			Msg string `json:"msg"`
		} `json:"errors"`
	}
	if json.Unmarshal(body, &apiErr) == nil && len(apiErr.Errors) > 0 {
		return fmt.Errorf("%s: %d %s", path, resp.StatusCode, apiErr.Errors[0].Msg)
	}
	return fmt.Errorf("%s: %d", path, resp.StatusCode)
}
