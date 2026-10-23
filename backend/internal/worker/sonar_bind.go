package worker

import (
	"context"
	"sort"
	"strings"
)

// BindReport is what BindExisting found, and did, for one project.
type BindReport struct {
	Project    string
	Repository string
	Status     string
	Failed     bool
}

// BindExisting binds the projects Forge created before it bound them
// ("<owner>_<repo>") to their repositories. With apply false it only reports
// what it would do.
//
// It never deletes. A "<key>2" project next to "<key>" is a duplicate that
// SonarCloud's UI created on import; SonarCloud refuses a second binding to
// the same repository, so the duplicate has to be deleted first.
func (s *SonarCloud) BindExisting(ctx context.Context, owner string, apply bool) ([]BindReport, error) {
	keys, err := s.projectKeys(ctx)
	if err != nil {
		return nil, err
	}
	sort.Strings(keys)
	exists := map[string]bool{}
	for _, k := range keys {
		exists[k] = true
	}
	orgID, err := s.organizationID(ctx)
	if err != nil {
		return nil, err
	}

	prefix := owner + "_"
	var reports []BindReport
	for _, key := range keys {
		if !strings.HasPrefix(key, prefix) {
			reports = append(reports, BindReport{Project: key, Status: "ignored: key does not follow <owner>_<repo>"})
			continue
		}
		original := strings.TrimSuffix(key, "2")
		if original != key && exists[original] {
			reports = append(reports, BindReport{Project: key, Status: "duplicate of " + original + ": delete it in SonarCloud"})
			continue
		}
		repo := owner + "/" + strings.TrimPrefix(key, prefix)
		r := BindReport{Project: key, Repository: repo}

		projectID, err := s.projectID(ctx, orgID, key)
		if err != nil {
			r.Status, r.Failed = err.Error(), true
			reports = append(reports, r)
			continue
		}
		slug, err := s.boundRepository(ctx, projectID)
		switch {
		case err != nil:
			r.Status, r.Failed = err.Error(), true
		case strings.EqualFold(slug, repo):
			r.Status = "already bound"
		case slug != "":
			r.Status, r.Failed = "bound to "+slug, true
		case exists[key+"2"]:
			r.Status, r.Failed = "blocked: delete "+key+"2 first", true
		case !apply:
			r.Status = "would bind"
		default:
			if err := s.BindRepository(ctx, key, repo); err != nil {
				r.Status, r.Failed = err.Error(), true
			} else {
				r.Status = "bound"
			}
		}
		reports = append(reports, r)
	}
	return reports, nil
}
