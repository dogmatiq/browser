package github

import (
	"context"
	"fmt"
	"net/url"
	"regexp"

	"github.com/dogmatiq/browser/components/askpass"
	"github.com/dogmatiq/browser/internal/githubapi"
	"github.com/dogmatiq/minibus"
	"github.com/google/go-github/v63/github"
)

// CredentialServer responds to requests for repository credentials.
type CredentialServer struct {
	Client *githubapi.AppClient
}

// Run starts the server.
func (s *CredentialServer) Run(ctx context.Context) error {
	minibus.Subscribe[askpass.Request](ctx)
	minibus.Ready(ctx)

	for req := range minibus.Inbox(ctx) {
		req := req.(askpass.Request)

		if !s.shouldRespond(req) {
			continue
		}

		token, err := s.getToken(ctx, req.RepoURL)
		if err != nil {
			return err
		}

		if err := minibus.Send(
			ctx,
			askpass.Response{
				CorrelationID: req.CorrelationID,
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
func (s *CredentialServer) shouldRespond(req askpass.Request) bool {
	if s.Client.BaseURL == nil {
		return req.RepoURL.Host == "github.com"
	}
	return req.RepoURL.Host == s.Client.BaseURL.Host
}

func (s *CredentialServer) getToken(
	ctx context.Context,
	repoURL *url.URL,
) (string, error) {
	matches := regexp.
		MustCompile(`^/([^/]+)/([^/.]+)`).
		FindStringSubmatch(repoURL.Path)

	if matches == nil {
		return "", fmt.Errorf("invalid repository URL: %s", repoURL)
	}

	owner := matches[1]
	repo := matches[2]

	in, _, err := s.Client.REST().Apps.FindRepositoryInstallation(ctx, owner, repo)
	if err != nil {
		return "", fmt.Errorf("unable to find installation for %s/%s repository: %w", owner, repo, err)
	}

	token, _, err := s.Client.REST().Apps.CreateInstallationToken(
		ctx,
		in.GetID(),
		&github.InstallationTokenOptions{
			Repositories: []string{repo},
			Permissions: &github.InstallationPermissions{
				Contents: github.String("read"),
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("unable to create installation token for %s/%s repository: %w", owner, repo, err)
	}

	return token.GetToken(), nil
}
