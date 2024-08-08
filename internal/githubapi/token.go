package githubapi

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"golang.org/x/oauth2"
)

// GenerateAppToken generates a GitHub App token for the app with the given
// client ID.
func GenerateAppToken(
	clientID string,
	privateKey *rsa.PrivateKey,
	expiresAt time.Time,
) (*oauth2.Token, error) {
	token, err := jwt.
		NewBuilder().
		Issuer(clientID).
		IssuedAt(time.Now()).
		Expiration(expiresAt).
		Build()

	if err != nil {
		return nil, fmt.Errorf("unable to build github app token claims: %w", err)
	}

	data, err := jwt.Sign(
		token,
		jwt.WithKey(
			jwa.RS256,
			privateKey,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to sign github app token: %w", err)
	}

	return &oauth2.Token{
		AccessToken: string(data),
		Expiry:      expiresAt,
	}, nil
}

// tokenSourceFunc adapts a function that returns a token into an
// [oauth2.TokenSource].
type tokenSourceFunc func() (*oauth2.Token, error)

// Token returns a new token.
func (f tokenSourceFunc) Token() (*oauth2.Token, error) {
	return f()
}
