package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-exec/tfexec"

	"github.com/murops/forge/backend/internal/requests"
)

type GitHubApp struct {
	ID             string
	InstallationID string
	PrivateKey     string
}

type AWSOptions struct {
	Enabled             bool
	Region              string
	OIDCProviderARN     string
	PlaceholderImageURI string
}

// Terraform state lives in Forge's Postgres, one workspace per repository.
type Terraform struct {
	ExecPath        string
	InfraDir        string
	StateConnString string
	Owner           string
	App             GitHubApp
	AWS             AWSOptions
	PrivateRulesets bool
	Sonar           *SonarCloud
	Log             *slog.Logger
}

func (t *Terraform) Provision(ctx context.Context, req requests.Request, tmpl requests.Template) (string, error) {
	tmp, err := os.MkdirTemp("", "forge-"+req.Name+"-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	if err := copyInfra(t.InfraDir, tmp); err != nil {
		return "", fmt.Errorf("copy infra: %w", err)
	}
	log := t.Log.With("request_id", req.ID, "workspace", req.Name)

	// before the repository exists, its first push already runs the sonar job
	if t.Sonar != nil {
		if req.Private {
			log.Warn("skipping SonarCloud, free plan only analyzes public projects")
		} else if err := t.Sonar.EnsureProject(ctx, t.Owner+"_"+req.Name, req.Name); err != nil {
			return "", fmt.Errorf("sonarcloud: %w", err)
		}
	}

	actionsVars := map[string]string{}
	if req.WithLambda {
		if !t.AWS.Enabled {
			log.Warn("lambda requested but AWS is disabled, creating repository only")
		} else {
			out, err := t.apply(ctx, log, filepath.Join(tmp, "stacks", "lambda"), req.Name, t.lambdaVars(req))
			if err != nil {
				return "", err
			}
			if err := json.Unmarshal(out["actions_variables"].Value, &actionsVars); err != nil {
				return "", fmt.Errorf("read lambda outputs: %w", err)
			}
		}
	}

	vars, err := t.repositoryVars(req, tmpl, actionsVars)
	if err != nil {
		return "", err
	}
	out, err := t.apply(ctx, log, filepath.Join(tmp, "stacks", "repository"), req.Name, vars)
	if err != nil {
		return "", err
	}
	var url string
	if err := json.Unmarshal(out["repository_url"].Value, &url); err != nil {
		return "", fmt.Errorf("read repository_url output: %w", err)
	}
	return url, nil
}

func (t *Terraform) apply(ctx context.Context, log *slog.Logger, dir, workspace string, vars map[string]any) (map[string]tfexec.OutputMeta, error) {
	if err := writeJSON(filepath.Join(dir, "forge.auto.tfvars.json"), vars); err != nil {
		return nil, err
	}
	tf, err := tfexec.NewTerraform(dir, t.ExecPath)
	if err != nil {
		return nil, err
	}
	if err := tf.SetEnv(t.env()); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	tf.SetStdout(&out)
	tf.SetStderr(&out)

	log = log.With("stack", filepath.Base(dir))
	run := func(step string, fn func() error) error {
		out.Reset()
		log.Info("terraform " + step)
		if err := fn(); err != nil {
			log.Error("terraform "+step+" failed", "output", tail(out.String(), 4000))
			return fmt.Errorf("terraform %s (%s): %w", step, filepath.Base(dir), err)
		}
		return nil
	}

	if err := run("init", func() error { return tf.Init(ctx) }); err != nil {
		return nil, err
	}
	if err := run("workspace", func() error { return selectWorkspace(ctx, tf, workspace) }); err != nil {
		return nil, err
	}
	if err := run("apply", func() error { return tf.Apply(ctx) }); err != nil {
		return nil, err
	}
	log.Info("terraform apply done", "summary", applySummary(out.String()))
	return tf.Output(ctx)
}

func (t *Terraform) repositoryVars(req requests.Request, tmpl requests.Template, actionsVars map[string]string) (map[string]any, error) {
	owner, repo, ok := strings.Cut(tmpl.SourceRepo, "/")
	if !ok {
		return nil, fmt.Errorf("template %s has invalid source repo %q", tmpl.Slug, tmpl.SourceRepo)
	}
	visibility := "public"
	if req.Private {
		visibility = "private"
	}
	checks := tmpl.RequiredChecks
	if checks == nil {
		checks = []string{}
	}
	return map[string]any{
		"name":                       req.Name,
		"description":                req.Description,
		"visibility":                 visibility,
		"github_owner":               t.Owner,
		"github_app_id":              t.App.ID,
		"github_app_installation_id": t.App.InstallationID,
		"github_app_private_key":     t.App.PrivateKey,
		"template":                   map[string]string{"owner": owner, "repository": repo},
		"template_topic":             tmpl.Slug,
		"private_rulesets_supported": t.PrivateRulesets,
		"required_checks":            checks,
		"actions_variables":          actionsVars,
	}, nil
}

func (t *Terraform) lambdaVars(req requests.Request) map[string]any {
	return map[string]any{
		"name":                     req.Name,
		"github_owner":             t.Owner,
		"aws_region":               t.AWS.Region,
		"github_oidc_provider_arn": t.AWS.OIDCProviderARN,
		"placeholder_image_uri":    t.AWS.PlaceholderImageURI,
	}
}

func (t *Terraform) env() map[string]string {
	env := map[string]string{}
	for _, kv := range os.Environ() {
		if k, v, ok := strings.Cut(kv, "="); ok {
			env[k] = v
		}
	}
	for _, k := range tfexec.ProhibitedEnv(env) {
		delete(env, k)
	}
	env["PG_CONN_STR"] = t.StateConnString
	return env
}

func selectWorkspace(ctx context.Context, tf *tfexec.Terraform, name string) error {
	existing, _, err := tf.WorkspaceList(ctx)
	if err != nil {
		return err
	}
	if slices.Contains(existing, name) {
		return tf.WorkspaceSelect(ctx, name)
	}
	return tf.WorkspaceNew(ctx, name)
}

func copyInfra(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			if d.Name() == ".terraform" || d.Name() == "tests" {
				return filepath.SkipDir
			}
			return os.MkdirAll(target, 0o700)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func writeJSON(path string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o600)
}

// applySummary returns the "Resources: N added, ..." part of the apply output.
func applySummary(out string) string {
	for line := range strings.Lines(out) {
		if _, after, ok := strings.Cut(line, "Apply complete! "); ok {
			return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(after), "."))
		}
	}
	return ""
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}

func LoadGitHubApp(id, installationID, keyFile string) (GitHubApp, error) {
	if id == "" || installationID == "" || keyFile == "" {
		return GitHubApp{}, errors.New("GITHUB_APP_ID, GITHUB_APP_INSTALLATION_ID and GITHUB_APP_PRIVATE_KEY_FILE are required")
	}
	key, err := os.ReadFile(keyFile)
	if err != nil {
		return GitHubApp{}, err
	}
	if len(bytes.TrimSpace(key)) == 0 {
		return GitHubApp{}, errors.New("GitHub App private key is empty")
	}
	return GitHubApp{ID: id, InstallationID: installationID, PrivateKey: string(key)}, nil
}
