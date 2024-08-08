package githubapi

import (
	"context"
	"crypto/rsa"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	githubrest "github.com/google/go-github/v63/github"
	githubgql "github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// AppClient is a client that accesses the GitHub APIs on behalf of a GitHub
// application.
type AppClient struct {
	ClientID   string
	PrivateKey *rsa.PrivateKey
	BaseURL    *url.URL
	Logger     *slog.Logger

	tokenID      atomic.Uint64
	activeTokens atomic.Int64

	initOnce, closeOnce sync.Once
	closed              chan struct{}
	rest                *githubrest.Client
	gql                 *githubgql.Client
}

// REST returns a client for the GitHub REST API.
func (c *AppClient) REST() *githubrest.Client {
	c.init()
	return c.rest
}

// GraphQL returns a client for the GitHub GraphQL API.
func (c *AppClient) GraphQL() *githubgql.Client {
	c.init()
	return c.gql
}

// InstallationClient creates a new client that accesses the GitHub APIs on
// behalf of a specific installation of a GitHub application.
func (c *AppClient) InstallationClient(id int64, opts *githubrest.InstallationTokenOptions) *InstallationClient {
	perms := opts.GetPermissions()
	if perms == nil || *perms == (githubrest.InstallationPermissions{}) {
		panic("must provide explicit permissions for installation token")
	}

	inst := &InstallationClient{
		parent:         c,
		InstallationID: id,
		TokenOptions:   opts,
		Logger:         c.Logger.With("installation_id", id),
		closed:         make(chan struct{}),
	}

	http := oauth2.NewClient(
		context.Background(),
		newTokenSource(inst.token),
	)

	inst.rest = newRESTClient(c.BaseURL, http)
	inst.gql = newGraphQLClient(c.BaseURL, http)

	return inst
}

// Close closes the client.
func (c *AppClient) Close() error {
	c.init()
	c.closeOnce.Do(func() {
		close(c.closed)
	})
	return nil
}

func (c *AppClient) init() {
	c.initOnce.Do(func() {
		http := oauth2.NewClient(
			context.Background(),
			newTokenSource(c.token),
		)
		c.rest = newRESTClient(c.BaseURL, http)
		c.gql = newGraphQLClient(c.BaseURL, http)

		c.closed = make(chan struct{})
	})
}

func (c *AppClient) token() (*oauth2.Token, error) {
	tokenID := c.tokenID.Add(1)
	expiresAt := time.Now().Add(tokenTTL)

	token, err := generateToken(
		tokenID,
		c.ClientID,
		c.PrivateKey,
		expiresAt,
	)
	if err != nil {
		return nil, err
	}

	c.Logger.Debug(
		"github application token generated",
		slog.Uint64("token_id", tokenID),
		slog.Int64("active_tokens", c.activeTokens.Add(1)),
		slog.Duration("token.ttl", tokenTTL),
		slog.Time("token.exp", expiresAt),
	)

	go func() {
		expired := time.NewTimer(time.Until(expiresAt))
		defer expired.Stop()

		select {
		case <-c.closed:
		case <-expired.C:
			c.Logger.Debug(
				"github application token expired",
				slog.Uint64("token_id", tokenID),
				slog.Int64("active_tokens", c.activeTokens.Add(-1)),
			)
		}
	}()

	return &oauth2.Token{
		AccessToken: token,
		Expiry:      expiresAt,
	}, nil
}

func newRESTClient(u *url.URL, h *http.Client) *githubrest.Client {
	if u == nil {
		return githubrest.NewClient(h)
	}

	rest, err := githubrest.
		NewClient(h).
		WithEnterpriseURLs(
			u.String(),
			u.String(),
		)
	if err != nil {
		panic(err)
	}

	return rest
}

func newGraphQLClient(u *url.URL, h *http.Client) *githubgql.Client {
	if u == nil {
		return githubgql.NewClient(h)
	}

	return githubgql.NewEnterpriseClient(u.String(), h)
}
