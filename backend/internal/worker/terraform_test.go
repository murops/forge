package worker

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/murops/forge/backend/internal/requests"
)

func newTerraform(aws AWSOptions) *Terraform {
	return &Terraform{
		Owner: "murops-forge",
		App:   GitHubApp{ID: "1", InstallationID: "2", PrivateKey: "key"},
		AWS:   aws,
		Log:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

var tmpl = requests.Template{Slug: "lambda-python", SourceRepo: "murops-forge/template-lambda-python", SupportsLambda: true}

func TestRepositoryVars(t *testing.T) {
	req := requests.Request{Name: "orders", Private: true}
	actions := map[string]string{"AWS_REGION": "sa-east-1"}

	v, err := newTerraform(AWSOptions{}).repositoryVars(req, tmpl, actions)
	if err != nil {
		t.Fatal(err)
	}
	if v["visibility"] != "private" || v["github_owner"] != "murops-forge" || v["template_topic"] != "lambda-python" {
		t.Errorf("unexpected vars: %v", v)
	}
	got := v["template"].(map[string]string)
	if got["owner"] != "murops-forge" || got["repository"] != "template-lambda-python" {
		t.Errorf("template = %v", got)
	}
	if checks := v["required_checks"].([]string); checks == nil || len(checks) != 0 {
		t.Errorf("required_checks must be an empty list, not null: %#v", checks)
	}
	if v["actions_variables"].(map[string]string)["AWS_REGION"] != "sa-east-1" {
		t.Errorf("actions variables not passed: %v", v["actions_variables"])
	}
}

func TestRepositoryVarsInvalidSourceRepo(t *testing.T) {
	_, err := newTerraform(AWSOptions{}).repositoryVars(requests.Request{Name: "orders"}, requests.Template{SourceRepo: "no-owner"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLambdaVars(t *testing.T) {
	aws := AWSOptions{Enabled: true, Region: "sa-east-1", OIDCProviderARN: "arn", PlaceholderImageURI: "img"}
	v := newTerraform(aws).lambdaVars(requests.Request{Name: "orders"})
	if v["name"] != "orders" || v["aws_region"] != "sa-east-1" || v["placeholder_image_uri"] != "img" {
		t.Errorf("unexpected vars: %v", v)
	}
}

func TestCopyInfraSkipsStateAndTests(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	for _, f := range []string{
		"stacks/repository/main.tf",
		"stacks/repository/.terraform/providers/big",
		"modules/github-repository/main.tf",
		"modules/github-repository/tests/repository.tftest.hcl",
	} {
		p := filepath.Join(src, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if err := copyInfra(src, dst); err != nil {
		t.Fatal(err)
	}

	for f, want := range map[string]bool{
		"stacks/repository/main.tf":         true,
		"modules/github-repository/main.tf": true,
		"stacks/repository/.terraform":      false,
		"modules/github-repository/tests":   false,
	} {
		_, err := os.Stat(filepath.Join(dst, f))
		if exists := err == nil; exists != want {
			t.Errorf("%s exists = %v, want %v", f, exists, want)
		}
	}
}

func TestApplySummary(t *testing.T) {
	out := "module.repository.github_branch.env[\"homol\"]: Creation complete\n\nApply complete! Resources: 8 added, 0 changed, 0 destroyed.\n\nOutputs:\n"
	if got := applySummary(out); got != "Resources: 8 added, 0 changed, 0 destroyed" {
		t.Errorf("got %q", got)
	}
	if got := applySummary("no changes"); got != "" {
		t.Errorf("got %q", got)
	}
}
