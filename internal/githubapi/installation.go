package githubapi

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v63/github"
	githubrest "github.com/google/go-github/v63/github"
	githubgql "github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// InstallationClient is a client that accesses the GitHub APIs on behalf of a
// specific installation of a GitHub application.
type InstallationClient struct {
	InstallationID int64
	TokenOptions   *githubrest.InstallationTokenOptions
	Logger         *slog.Logger

	parent *AppClient
	rest   *githubrest.Client
	gql    *githubgql.Client

	closeOnce sync.Once
	closed    chan struct{}
}

// REST returns a client for the GitHub REST API.
func (c *InstallationClient) REST() *githubrest.Client {
	return c.rest
}

// GraphQL returns a client for the GitHub GraphQL API.
func (c *InstallationClient) GraphQL() *githubgql.Client {
	return c.gql
}

// Close closes the client by revoking the installation token.
func (c *InstallationClient) Close() error {
	c.closeOnce.Do(func() {
		close(c.closed)
	})
	return nil
}

func (c *InstallationClient) token() (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tokenID := c.parent.tokenID.Add(1)
	token, _, err := c.parent.REST().Apps.CreateInstallationToken(
		ctx,
		c.InstallationID,
		c.TokenOptions,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create github installation token: %w", err)
	}

	expiresAt := token.GetExpiresAt().Time

	c.Logger.Debug(
		"github installation token generated",
		slog.Uint64("token_id", tokenID),
		slog.Int64("active_tokens", c.parent.activeTokens.Add(1)),
		slog.String("repositories", repositoriesAsString(token.Repositories)),
		slog.String("permissions", permissionsAsString(token.GetPermissions())),
		slog.Duration("expires_in", time.Until(expiresAt)),
		slog.Time("expires_at", expiresAt),
	)

	go func() {
		logExpired := func() {
			c.Logger.Debug(
				"github installation token expired",
				slog.Uint64("token_id", tokenID),
				slog.Int64("active_tokens", c.parent.activeTokens.Add(-1)),
			)
		}

		expired := time.NewTimer(time.Until(expiresAt))
		defer expired.Stop()

		select {
		case <-c.parent.closed:
			c.revokeToken(tokenID, token)

		case <-c.closed:
			if !c.revokeToken(tokenID, token) {
				select {
				case <-c.parent.closed:
				case <-expired.C:
					logExpired()
				}
			}

		case <-expired.C:
			logExpired()
		}

	}()

	return &oauth2.Token{
		AccessToken: token.GetToken(),
		Expiry:      token.GetExpiresAt().Time,
	}, nil
}

func (c *InstallationClient) revokeToken(tokenID uint64, token *github.InstallationToken) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rest := newRESTClient(c.parent.BaseURL, nil).
		WithAuthToken(token.GetToken())

	if _, err := rest.Apps.RevokeInstallationToken(ctx); err != nil {
		c.Logger.Warn(
			"unable to revoke github installation token",
			slog.Uint64("token_id", tokenID),
			slog.Int64("active_tokens", c.parent.activeTokens.Load()),
			slog.String("error", err.Error()),
		)

		return false
	}

	c.Logger.Debug(
		"github installation token revoked",
		slog.Uint64("token_id", tokenID),
		slog.Int64("active_tokens", c.parent.activeTokens.Add(-1)),
	)

	return true
}

func repositoriesAsString(repositories []*githubrest.Repository) string {
	if len(repositories) == 0 {
		return "(all)"
	}

	var repos []string
	for _, repo := range repositories {
		repos = append(repos, repo.GetName())
	}

	slices.Sort(repos)

	return strings.Join(repos, ", ")
}

func permissionsAsString(p *githubrest.InstallationPermissions) string {
	var perms []string

	v := reflect.ValueOf(p).Elem()
	t := v.Type()

	for i := range t.NumField() {
		field := t.Field(i)
		value := v.Field(i)

		if value.IsNil() {
			continue
		}

		tag := field.Tag.Get("json")
		name := strings.SplitN(tag, ",", 2)[0]

		perms = append(
			perms,
			fmt.Sprintf(
				"%s:%s",
				name,
				value.Elem().Interface(),
			),
		)
	}

	slices.Sort(perms)

	return strings.Join(perms, ", ")
}
