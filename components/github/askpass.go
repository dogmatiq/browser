package github

import (
	"context"
	"log/slog"
	"net/url"
	"regexp"
	"time"

	"github.com/dogmatiq/browser/components/askpass"
	"github.com/dogmatiq/browser/internal/githubapi"
	"github.com/dogmatiq/browser/messages"
	"github.com/dogmatiq/minibus"
	"golang.org/x/oauth2"
)

// AskpassServer responds to requests for repository credentials.
type AskpassServer struct {
	Client *githubapi.AppClient
	Logger *slog.Logger

	reposByID   map[string]*askpassRepo
	reposByName map[string]*askpassRepo
}

type askpassRepo struct {
	ID          int64
	Name        string
	TokenSource oauth2.TokenSource
}

// Run starts the server.
func (s *AskpassServer) Run(ctx context.Context) error {
	minibus.Subscribe[messages.RepoFound](ctx)
	minibus.Subscribe[messages.RepoLost](ctx)
	minibus.Subscribe[askpass.Request](ctx)
	minibus.Ready(ctx)

	s.reposByID = map[string]*askpassRepo{}
	s.reposByName = map[string]*askpassRepo{}

	for m := range minibus.Inbox(ctx) {
		if err := s.handleMessage(ctx, m); err != nil {
			return err
		}
	}

	return nil
}

func (s *AskpassServer) handleMessage(
	ctx context.Context,
	m any,
) error {
	switch m := m.(type) {
	case messages.RepoFound:
		return s.handleRepoFound(ctx, m)
	case messages.RepoLost:
		return s.handleRepoLost(ctx, m)
	case askpass.Request:
		return s.handleAskpassRequest(ctx, m)
	default:
		panic("unexpected message type")
	}
}

func (s *AskpassServer) handleRepoFound(
	ctx context.Context,
	m messages.RepoFound,
) error {
	if m.RepoSource != repoSource(s.Client) {
		return nil
	}

	repoID, err := unmarshalRepoID(m.RepoID)
	if err != nil {
		return err
	}

	in, _, err := s.Client.REST().Apps.FindRepositoryInstallationByID(ctx, repoID)
	if err != nil {
		return err
	}

	r := &askpassRepo{
		ID:          repoID,
		Name:        m.RepoName,
		TokenSource: s.Client.InstallationTokenSource(in.GetID()),
	}

	s.reposByID[m.RepoID] = r
	s.reposByName[m.RepoName] = r

	return nil
}

func (s *AskpassServer) handleRepoLost(
	_ context.Context,
	m messages.RepoLost,
) error {
	if m.RepoSource != repoSource(s.Client) {
		return nil
	}

	repo := s.reposByID[m.RepoID]
	delete(s.reposByID, m.RepoID)
	delete(s.reposByName, repo.Name)

	return nil
}

func (s *AskpassServer) handleAskpassRequest(
	ctx context.Context,
	m askpass.Request,
) (err error) {
	start := time.Now()
	defer func() {
		s.Logger.DebugContext(
			ctx,
			"askpass request handled",
			slog.Duration("elapsed", time.Since(start)),
			slog.Any("error", err),
		)
	}()

	repoName, ok := s.parseRepoURL(m.RepoURL)
	if !ok {
		return nil
	}

	repo, ok := s.reposByName[repoName]
	if !ok {
		return nil
	}

	token, err := repo.TokenSource.Token()
	if err != nil {
		return err
	}

	return minibus.Send(
		ctx,
		askpass.Response{
			CorrelationID: m.CorrelationID,
			RepoSource:    repoSource(s.Client),
			RepoID:        marshalRepoID(repo.ID),
			RepoURL:       m.RepoURL,
			Username:      "x-access-token",
			Password:      token.AccessToken,
		},
	)
}

func (s *AskpassServer) parseRepoURL(u *url.URL) (string, bool) {
	if s.Client.BaseURL == nil {
		if u.Host != "github.com" {
			return "", false
		}
	} else if u.Host != s.Client.BaseURL.Host {
		return "", false
	}

	matches := regexp.
		MustCompile(`^/([^/]+/[^/.]+)`).
		FindStringSubmatch(u.Path)

	if matches == nil {
		return "", false
	}

	return matches[1], true
}
