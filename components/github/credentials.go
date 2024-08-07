package github

import (
	"context"
	"fmt"
	"net/url"
	"regexp"

	"github.com/dogmatiq/browser/internal/githubutils"
	"github.com/dogmatiq/browser/messages"
	"github.com/dogmatiq/minibus"
	"github.com/google/go-github/v63/github"
)

type CredentialServer struct {
	Clients *githubutils.ClientSet

	tokens map[string]*github.InstallationToken
}

func (s *CredentialServer) Run(ctx context.Context) error {
	minibus.Subscribe[messages.RepoCredentialsRequest](ctx)
	minibus.Ready(ctx)

	for m := range minibus.Inbox(ctx) {
		m := m.(messages.RepoCredentialsRequest)

		u, err := url.Parse(m.RepoURL)
		if err != nil {
			return fmt.Errorf("cannot parse repository URL: %w", err)
		}

		if !s.shouldRespond(u) {
			continue
		}

		token, err := s.getToken(ctx, u)
		if err != nil {
			return err
		}

		if err := minibus.Send(
			ctx,
			messages.RepoCredentialsResponse{
				CorrelationID: m.CorrelationID,
				Username:      "x-access-token",
				Password:      token,
			},
		); err != nil {
			return err
		}
	}

	return nil
}

// shouldRespond returns true if the server should respond to a request for
// credentials for the given repository URL.
func (s *CredentialServer) shouldRespond(repoURL *url.URL) bool {
	baseURL := s.Clients.App().BaseURL

	if repoURL.Host == "github.com" && baseURL.Host == "api.github.com" {
		return true
	}

	if repoURL.Host == baseURL.Host {
		return true
	}

	return false
}

func (s *CredentialServer) getToken(
	ctx context.Context,
	repoURL *url.URL,
) (string, error) {
	matches := regexp.
		MustCompile(`^/([^/]+)/([^/.]+)`).
		FindStringSubmatch(repoURL.Path)

	if matches == nil {
		return "", fmt.Errorf("github: invalid repository URL: %s", repoURL)
	}

	owner := matches[1]
	repo := matches[2]

	in, _, err := s.Clients.App().Apps.FindRepositoryInstallation(ctx, owner, repo)
	if err != nil {
		return "", fmt.Errorf("github: unable to find installation for %s/%s repository: %w", owner, repo, err)
	}

	token, _, err := s.Clients.App().Apps.CreateInstallationToken(
		ctx,
		in.GetID(),
		&github.InstallationTokenOptions{
			// 	// Repositories: []string{repo},
			Permissions: &github.InstallationPermissions{
				Contents: github.String("read"),
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("github: unable to create installation token for %s/%s repository: %w", owner, repo, err)
	}

	return token.GetToken(), nil
}
