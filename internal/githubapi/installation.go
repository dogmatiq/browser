package githubapi

import (
	githubrest "github.com/google/go-github/v63/github"
	githubgql "github.com/shurcooL/githubv4"
)

// InstallationClient is a client that accesses the GitHub APIs on behalf of a
// specific installation of a GitHub application.
type InstallationClient struct {
	rest *githubrest.Client
	gql  *githubgql.Client
}

// REST returns a client for the GitHub REST API.
func (c *InstallationClient) REST() *githubrest.Client {
	return c.rest
}

// GraphQL returns a client for the GitHub GraphQL API.
func (c *InstallationClient) GraphQL() *githubgql.Client {
	return c.gql
}
