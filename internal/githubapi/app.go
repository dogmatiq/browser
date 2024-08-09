package githubapi

import (
	"context"
	"crypto/rsa"
	"log/slog"
	"net/http"
	"net/url"
	"sync"

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

	initOnce sync.Once
	rest     *githubrest.Client
	gql      *githubgql.Client

	clients      sync.Map // map[int64]*InstallationClient
	tokenSources sync.Map // map[int64]oauth2.TokenSource
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
func (c *AppClient) InstallationClient(id int64) *InstallationClient {
	cli, ok := c.clients.Load(id)

	if !ok {
		http := oauth2.NewClient(
			context.Background(),
			c.InstallationTokenSource(id),
		)

		cli, _ = c.clients.LoadOrStore(
			id,
			&InstallationClient{
				rest: newRESTClient(c.BaseURL, http),
				gql:  newGraphQLClient(c.BaseURL, http),
			},
		)
	}

	return cli.(*InstallationClient)
}

// InstallationTokenSource returns a token source that creates installation
// tokens for a specific installation of a GitHub application.
func (c *AppClient) InstallationTokenSource(id int64) oauth2.TokenSource {
	ts, ok := c.tokenSources.Load(id)

	if !ok {
		ts, _ = c.tokenSources.LoadOrStore(
			id,
			oauth2.ReuseTokenSource(
				nil,
				&installationTokenSource{
					Client:         c,
					InstallationID: id,
				},
			),
		)
	}

	return ts.(oauth2.TokenSource)
}

func (c *AppClient) init() {
	c.initOnce.Do(func() {
		http := oauth2.NewClient(
			context.Background(),
			&appTokenSource{
				ClientID:   c.ClientID,
				PrivateKey: c.PrivateKey,
			},
		)
		c.rest = newRESTClient(c.BaseURL, http)
		c.gql = newGraphQLClient(c.BaseURL, http)
	})
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
