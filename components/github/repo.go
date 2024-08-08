package github

import (
	"github.com/google/go-github/v63/github"
)

// isIgnored returns true if the repository should be ignored.
func isIgnored(r *github.Repository) bool {
	return r.GetIsTemplate() ||
		r.GetArchived() ||
		r.GetFork()
}
