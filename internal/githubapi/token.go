package githubapi

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"golang.org/x/oauth2"
)

// appTokenSource is an [oauth2.TokenSource] that creates application tokens for
// a GitHub application.
type appTokenSource struct {
	ClientID   string
	PrivateKey *rsa.PrivateKey
}

// Token returns a new application token.
func (s *appTokenSource) Token() (*oauth2.Token, error) {
	expiresAt := time.Now().Add(1 * time.Minute)

	token, err := jwt.
		NewBuilder().
		Issuer(s.ClientID).
		IssuedAt(time.Now()).
		Expiration(expiresAt).
		Build()
	if err != nil {
		return nil, fmt.Errorf("unable to generate github application token: %w", err)
	}

	data, err := jwt.Sign(
		token,
		jwt.WithKey(
			jwa.RS256,
			s.PrivateKey,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to sign github application token: %w", err)
	}

	return &oauth2.Token{
		AccessToken: string(data),
		Expiry:      expiresAt,
	}, nil
}

// installationTokenSource is an [oauth2.TokenSource] that creates installation
// tokens for a GitHub application.
type installationTokenSource struct {
	Client         *AppClient
	InstallationID int64
}

// Token returns a new installation token.
func (s *installationTokenSource) Token() (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	token, _, err := s.Client.REST().Apps.CreateInstallationToken(ctx, s.InstallationID, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create github installation token: %w", err)
	}

	return &oauth2.Token{
		AccessToken: token.GetToken(),
		Expiry:      token.GetExpiresAt().Time,
	}, nil
}
