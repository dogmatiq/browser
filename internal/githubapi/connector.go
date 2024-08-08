package githubapi

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/url"
	"time"

	githubrest "github.com/google/go-github/v63/github"
	githubgql "github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// Connector is used to create GitHub API clients authenticated as a GitHub
// application.
type Connector struct {
	ClientID   string
	PrivateKey *rsa.PrivateKey
	BaseURL    *url.URL
}

// WithApp calls fn with a client that is authenticated as the GitHub App.
func (c *Connector) WithApp(
	ctx context.Context,
	fn func(context.Context, *AppClient) error,
) error {
	rest, gql := c.newClients(
		ctx,
		func() (*oauth2.Token, error) {
			return GenerateAppToken(
				c.ClientID,
				c.PrivateKey,
				time.Now().Add(30*time.Second),
			)
		},
	)

	cli := &AppClient{rest, gql}
	defer cli.Close()

	return fn(ctx, cli)
}

// WithInstallation calls fn with a client that is authenticated as the GitHub
// App and has an installation token for the given installation ID.
//
// The installation token is revoked when fn returns.
func (c *Connector) WithInstallation(
	ctx context.Context,
	instID int64,
	opts *githubrest.InstallationTokenOptions,
	fn func(context.Context, *InstallationClient) error,
) error {
	perms := opts.GetPermissions()
	if perms == nil || *perms == (githubrest.InstallationPermissions{}) {
		panic("must provide explicit permissions for installation token")
	}

	return c.WithApp(
		ctx,
		func(ctx context.Context, app *AppClient) error {
			token, _, err := app.REST.Apps.CreateInstallationToken(
				ctx,
				instID,
				opts,
			)
			if err != nil {
				return fmt.Errorf("unable to create installation token: %w", err)
			}

			ctx, cancel := context.WithDeadline(ctx, token.GetExpiresAt().Time)
			defer cancel()

			rest, gql := c.newClients(
				ctx,
				func() (*oauth2.Token, error) {
					return &oauth2.Token{
						AccessToken: token.GetToken(),
						Expiry:      token.GetExpiresAt().Time,
					}, nil
				},
			)

			cli := &InstallationClient{
				rest,
				gql,
				instID,
				token.GetToken(),
			}
			defer cli.Close()

			return fn(ctx, cli)
		},
	)
}

func (c *Connector) newClients(
	ctx context.Context,
	ts func() (*oauth2.Token, error),
) (*githubrest.Client, *githubgql.Client) {
	http := oauth2.NewClient(ctx, tokenSourceFunc(ts))
	rest := githubrest.NewClient(http)
	gql := githubgql.NewClient(http)

	if c.BaseURL != nil {
		var err error
		rest, err = rest.WithEnterpriseURLs(
			c.BaseURL.String(),
			c.BaseURL.String(),
		)
		if err != nil {
			panic(err)
		}

		gql = githubgql.NewEnterpriseClient(
			c.BaseURL.String(),
			http,
		)
	}

	return rest, gql
}

// WithApp calls fn with a client that is authenticated as the GitHub App.
func WithApp[T any](
	ctx context.Context,
	conn *Connector,
	fn func(context.Context, *AppClient) (T, error),
) (result T, _ error) {
	return result, conn.WithApp(
		ctx,
		func(ctx context.Context, cli *AppClient) (err error) {
			result, err = fn(ctx, cli)
			return
		},
	)
}

// WithInstallation calls fn with a client that is authenticated as the GitHub
// App and has an installation token for the given installation ID.
//
// The installation token is revoked when fn returns.
func WithInstallation[T any](
	ctx context.Context,
	conn *Connector,
	instID int64,
	opts *githubrest.InstallationTokenOptions,
	fn func(context.Context, *InstallationClient) (T, error),
) (result T, _ error) {
	return result, conn.WithInstallation(
		ctx,
		instID,
		opts,
		func(ctx context.Context, cli *InstallationClient) (err error) {
			result, err = fn(ctx, cli)
			return
		},
	)
}
