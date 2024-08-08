package githubapi

import (
	"context"
	"time"

	githubrest "github.com/google/go-github/v63/github"
	githubgql "github.com/shurcooL/githubv4"
)

// AppClient is a client that accesses the GitHub APIs on behalf of a GitHub
// application.
type AppClient struct {
	REST    *githubrest.Client
	GraphQL *githubgql.Client
}

// Close closes the client.
func (c *AppClient) Close() error {
	return nil
}

// InstallationClient is a client that accesses the GitHub APIs on behalf of a
// specific installation of a GitHub application.
type InstallationClient struct {
	REST              *githubrest.Client
	GraphQL           *githubgql.Client
	InstallationID    int64
	InstallationToken string
}

// Close closes the client by revoking the installation token.
func (c *InstallationClient) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.REST.Apps.RevokeInstallationToken(ctx)
	return err
}
