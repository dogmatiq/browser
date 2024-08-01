package githubx

import (
	"context"
	"crypto/rsa"
	"net/http"
	"net/url"

	"github.com/google/go-github/v38/github"
	"golang.org/x/oauth2"
)

// Connector creates GitHub clients with different authentication credentials on
// behalf of a GitHub application.
type Connector struct {
	// AppClient is a GitHub client that is setup to authenticate using the
	// GitHub application's credentials.
	AppClient *github.Client

	// OAuthConfig is the OAuth configuration for the application. It may be nil
	// if the application does not authenticate or act on behalf of GitHub
	// users.
	OAuthConfig *oauth2.Config

	// Permissions optionally limits the client's permissions to a subset of
	// those available to the application.
	//
	// If it is nil, the full set of permissions granted to the application are
	// granted to each new client.
	Permissions *github.InstallationPermissions

	// Transport is the HTTP transport used by GitHub clients created by the
	// connector. If it is nil the default transport is used.
	Transport http.RoundTripper
}

// NewConnector returns a new connector.
//
// This is a convenience function that configures the connector with useful
// defaults.
//
// clientID is the OAuth client ID. If it is empty the OAuth configuration is
// omitted.
//
// baseURL is the base URL for the API of a GitHub Enterprise Server
// installation. If it is nil or empty the connector uses github.com.
//
// transport is the HTTP transport used by the GitHub clients created by the
// connector. If it is nil the default HTTP transport is used.
func NewConnector(
	appID int64,
	appKey *rsa.PrivateKey,
	clientID string,
	clientSecret string,
	baseURL *url.URL,
	transport http.RoundTripper,
) (*Connector, error) {
	hc := oauth2.NewClient(
		contextWithTransport(transport),
		&AppTokenSource{
			AppID:      appID,
			PrivateKey: appKey,
		},
	)

	c := &Connector{
		Transport: transport,
	}

	if baseURL == nil || baseURL.String() == "" {
		c.AppClient = github.NewClient(hc)
	} else {
		var err error
		c.AppClient, err = github.NewEnterpriseClient(
			baseURL.String(),
			baseURL.String(),
			hc,
		)
		if err != nil {
			return nil, err
		}
	}

	if clientID != "" {
		c.OAuthConfig = &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     NewOAuthEndpoint(c.AppClient.BaseURL),
		}
	}

	return c, nil
}

// InstallationClient returns a GitHub client that is authenticated as a
// specific installation of a GitHub application.
func (c *Connector) InstallationClient(ctx context.Context, id int64) (*github.Client, error) {
	return c.newClient(
		id,
		&github.InstallationTokenOptions{
			Permissions: c.Permissions,
		},
	), nil
}

// RepositoryClient returns a GitHub client that is authenticated as a the
// installation of a GitHub application that grants the application access to a
// specific repository.
//
// The client uses an access token that is only granted access to the specified
// repository.
func (c *Connector) RepositoryClient(ctx context.Context, id int64) (*github.Client, bool, error) {
	i, res, err := c.AppClient.Apps.FindRepositoryInstallationByID(ctx, id)
	if err != nil {
		if res.StatusCode == http.StatusNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}

	return c.newClient(
		i.GetID(),
		&github.InstallationTokenOptions{
			RepositoryIDs: []int64{id},
			Permissions:   c.Permissions,
		},
	), true, nil
}

// UserClient returns a GitHub client that is authenticated as a GitHub user who
// has granted the application permission to act on their behalf.
func (c *Connector) UserClient(ctx context.Context, t *oauth2.Token) (*github.Client, error) {
	client := github.NewClient(
		c.OAuthConfig.Client(
			contextWithTransport(c.Transport),
			t,
		),
	)

	c.applySettings(client)

	return client, nil
}

// newClient returns a GitHub client for a specific installation.
func (c *Connector) newClient(
	installationID int64,
	options *github.InstallationTokenOptions,
) *github.Client {
	client := github.NewClient(
		oauth2.NewClient(
			contextWithTransport(c.Transport),
			&InstallationTokenSource{
				AppClient:      c.AppClient,
				InstallationID: installationID,
				Options:        options,
			},
		),
	)

	c.applySettings(client)

	return client
}

// contextWithTransport returns a context containing a HTTP client that uses t.
// This context is the only way to provide the appropriate HTTP client to the
// oauth2 package.
func contextWithTransport(t http.RoundTripper) context.Context {
	ctx := context.Background()

	if t != nil {
		// oauth2.NewClient() pulls the transport from a client passed via the
		// context, but it does not use any other fields from the client.
		ctx = context.WithValue(
			ctx,
			oauth2.HTTPClient,
			&http.Client{
				Transport: t,
			},
		)
	}

	return ctx
}

// applySettings copies settings from c.AppClient client to the given client.
func (c *Connector) applySettings(client *github.Client) {
	client.BaseURL = c.AppClient.BaseURL
	client.UploadURL = c.AppClient.UploadURL
	client.UserAgent = c.AppClient.UserAgent
}
