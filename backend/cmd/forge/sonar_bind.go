package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"

	"github.com/murops/forge/backend/internal/worker"
)

// runSonarBind binds the SonarCloud projects of repositories created before
// Forge bound them. It is a dry run unless -apply is given, and never deletes.
func runSonarBind(args []string) int {
	fs := flag.NewFlagSet("sonar-bind", flag.ContinueOnError)
	apply := fs.Bool("apply", false, "bind the projects (default: only show what would happen)")
	owner := fs.String("owner", envOr("GITHUB_ORG", "murops-forge"), "GitHub organization of the repositories")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: forge sonar-bind [-apply] [-owner org]\n\nNeeds SONAR_TOKEN (organization admin) and SONAR_ORGANIZATION.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	token, org := os.Getenv("SONAR_TOKEN"), os.Getenv("SONAR_ORGANIZATION")
	if token == "" || org == "" {
		fmt.Fprintln(os.Stderr, "SONAR_TOKEN and SONAR_ORGANIZATION are required")
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	reports, err := worker.NewSonarCloud(token, org).BindExisting(ctx, *owner, *apply)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sonar-bind:", err)
		return 1
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "PROJECT\tREPOSITORY\tSTATUS")
	failed := false
	for _, r := range reports {
		repo := r.Repository
		if repo == "" {
			repo = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.Project, repo, r.Status)
		failed = failed || r.Failed
	}
	w.Flush()
	if !*apply {
		fmt.Println("\ndry run: nothing changed. Run with -apply to bind.")
	}
	if failed {
		return 1
	}
	return 0
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
