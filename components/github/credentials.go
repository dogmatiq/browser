package github

import (
	"context"
	"fmt"
	"net/url"
	"regexp"

	"github.com/dogmatiq/browser/internal/githubapi"
	"github.com/dogmatiq/browser/messages"
	"github.com/dogmatiq/minibus"
	"github.com/google/go-github/v63/github"
)

// CredentialServer responds to requests for repository credentials.
type CredentialServer struct {
	Connector *githubapi.Connector
}

// Run starts the server.
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
	if s.Connector.BaseURL == nil {
		return repoURL.Host == "github.com"
	}
	return repoURL.Host == s.Connector.BaseURL.Host
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

	return githubapi.WithApp(
		ctx,
		s.Connector,
		func(ctx context.Context, c *githubapi.AppClient) (string, error) {
			in, _, err := c.REST.Apps.FindRepositoryInstallation(ctx, owner, repo)
			if err != nil {
				return "", fmt.Errorf("unable to find installation for %s/%s repository: %w", owner, repo, err)
			}

			token, _, err := c.REST.Apps.CreateInstallationToken(
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
		},
	)
}
