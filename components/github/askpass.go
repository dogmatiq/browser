package github

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"time"

	"github.com/dogmatiq/browser/components/askpass"
	"github.com/dogmatiq/browser/internal/githubapi"
	"github.com/dogmatiq/minibus"
	"github.com/google/go-github/v63/github"
)

// AskpassServer responds to requests for repository credentials.
type AskpassServer struct {
	Client *githubapi.AppClient
	Logger *slog.Logger

	cache map[string]*github.InstallationToken // owner (org/user) -> token
}

// Run starts the server.
func (s *AskpassServer) Run(ctx context.Context) error {
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
				RepoURL:       req.RepoURL,
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
func (s *AskpassServer) shouldRespond(req askpass.Request) bool {
	if s.Client.BaseURL == nil {
		return req.RepoURL.Host == "github.com"
	}
	return req.RepoURL.Host == s.Client.BaseURL.Host
}

func (s *AskpassServer) getToken(
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

	token, ok := s.cache[owner]

	if ok {
		if time.Until(token.ExpiresAt.Time) > 1*time.Minute {
			s.Logger.DebugContext(
				ctx,
				"reused existing installation token for askpass",
				slog.String("repo.url", repoURL.String()),
				slog.Duration("token.ttl", time.Until(token.ExpiresAt.Time)),
				slog.Time("token.exp", token.ExpiresAt.Time),
			)

			return token.GetToken(), nil
		}

		delete(s.cache, owner)
	}

	if s.cache == nil {
		s.cache = map[string]*github.InstallationToken{}
	}

	token, err := s.generateToken(ctx, owner, repo)
	if err != nil {
		return "", err
	}

	s.cache[owner] = token

	s.Logger.DebugContext(
		ctx,
		"generated installation token for askpass",
		slog.String("repo.url", repoURL.String()),
		slog.Duration("token.ttl", time.Until(token.ExpiresAt.Time)),
		slog.Time("token.exp", token.ExpiresAt.Time),
	)

	return token.GetToken(), nil
}

func (s *AskpassServer) generateToken(
	ctx context.Context,
	owner, repo string,
) (*github.InstallationToken, error) {
	in, _, err := s.Client.REST().Apps.FindRepositoryInstallation(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("unable to find installation for %s/%s repository: %w", owner, repo, err)
	}

	token, _, err := s.Client.REST().Apps.CreateInstallationToken(
		ctx,
		in.GetID(),
		&github.InstallationTokenOptions{
			Permissions: &github.InstallationPermissions{
				Contents: github.String("read"),
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create installation token for %s/%s repository: %w", owner, repo, err)
	}

	return token, nil
}
