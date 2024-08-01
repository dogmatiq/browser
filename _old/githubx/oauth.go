package githubx

import (
	"net/url"

	"golang.org/x/oauth2"
)

// NewOAuthEndpoint returns the OAuth endpoint configuration to use for the
// GitHub installation at the given URL.
//
// If u is empty, the configuration for github.com is returned.
func NewOAuthEndpoint(u *url.URL) oauth2.Endpoint {
	ep := oauth2.Endpoint{
		AuthURL:   "https://github.com/login/oauth/authorize",
		TokenURL:  "https://github.com/login/oauth/access_token",
		AuthStyle: oauth2.AuthStyleInParams,
	}

	if u == nil ||
		u.String() == "" ||
		u.Hostname() == "github.com" ||
		u.Hostname() == "api.github.com" {
		return ep
	}

	// Clone the URL so we can manipulate it's path without messing up the
	// GitHub client.
	u, _ = url.Parse(u.String()) // clone

	u.Path = "/login/oauth/authorize"
	ep.AuthURL = u.String()

	u.Path = "/login/oauth/access_token"
	ep.TokenURL = u.String()

	return ep
}
