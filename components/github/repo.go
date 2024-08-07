package github

import (
	"strconv"

	"github.com/google/go-github/v63/github"
	"github.com/google/uuid"
)

// generateRepoID returns a unique identifier for the repository.
func generateRepoID(r *github.Repository) uuid.UUID {
	id := strconv.FormatInt(r.GetID(), 10)
	return uuid.NewSHA1(repoIDNamespace, []byte(id))
}

var repoIDNamespace = uuid.MustParse("a3be9e52-3e52-4824-8c40-bc42c7c31bff")

// isIgnored returns true if the repository should be ignored.
func isIgnored(r *github.Repository) bool {
	return r.GetIsTemplate() ||
		r.GetArchived() ||
		r.GetFork()
}
