package github

import (
	"fmt"
	"strconv"

	"github.com/dogmatiq/browser/internal/githubapi"
	"github.com/google/go-github/v63/github"
)

func repoSource(c *githubapi.AppClient) string {
	if c.BaseURL == nil {
		return "github"
	}
	return fmt.Sprintf("github@%s", c.BaseURL.Host)
}

func marshalRepoID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func unmarshalRepoID(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// isIgnored returns true if the repository should be ignored.
func isIgnored(r *github.Repository) bool {
	return r.GetIsTemplate() ||
		r.GetArchived() ||
		r.GetFork()
}
