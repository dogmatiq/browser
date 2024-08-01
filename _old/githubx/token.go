package githubx

import (
	"context"
	"crypto/rsa"
	"strconv"
	"time"

	"github.com/dogmatiq/linger"
	"github.com/google/go-github/v38/github"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jws" // nolint // TODO https://github.com/dogmatiq/browser/issues/1
)

// AppTokenSource is an implementation of oauth2.TokenSource that generates
// GitHub API tokens that authenticate as a GitHub application.
type AppTokenSource struct {
	// AppID is the GitHub application ID.
	AppID int64

	// PrivateKey is the application's private key.
	PrivateKey *rsa.PrivateKey

	// TTL is the default amount of time that each token remains valid.
	// If it is non-positive, a value of 1 minute is used.
	TTL time.Duration
}

// Token returns an OAuth token that authenticates as the GitHub application.
func (s *AppTokenSource) Token() (*oauth2.Token, error) {
	ttl := linger.MustCoalesce(s.TTL, 1*time.Minute)
	expiresAt := time.Now().Add(ttl)

	header := &jws.Header{
		Algorithm: "RS256",
		Typ:       "JWT",
	}

	claims := &jws.ClaimSet{
		Iss: strconv.FormatInt(s.AppID, 10),
		Exp: expiresAt.Unix(),
	}

	// TODO: replace use of deprecated jws package with something else.
	token, err := jws.Encode(header, claims, s.PrivateKey)
	if err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken: token,
		Expiry:      expiresAt,
	}, nil
}

// InstallationTokenSource is an implementation of oauth2.TokenSource that
// generates GitHub API tokens that authenticate a specific installation of a
// GitHub application.
type InstallationTokenSource struct {
	// AppClient is a GitHub client that is authenticated as the application
	// itself. See AppTokenSource.
	AppClient *github.Client

	// InstallationID is the ID of the installation.
	InstallationID int64

	// Options is a set of options to use when generating installation-specific
	// tokens.
	Options *github.InstallationTokenOptions

	// RequestTimeout is the amount of time to allow for token generation via
	// the GitHub client. If it is non-positive, a value of 10 seconds is used.
	RequestTimeout time.Duration
}

// Token returns an OAuth token that authenticates as a specific installation of
// a GitHub application.
func (s *InstallationTokenSource) Token() (*oauth2.Token, error) {
	ctx, cancel := linger.ContextWithTimeout(
		context.Background(),
		s.RequestTimeout,
		10*time.Second,
	)
	defer cancel()

	token, _, err := s.AppClient.Apps.CreateInstallationToken(
		ctx,
		s.InstallationID,
		s.Options,
	)
	if err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken: token.GetToken(),
		Expiry:      token.GetExpiresAt(),
	}, nil
}
