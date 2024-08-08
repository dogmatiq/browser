package githubapi

import (
	"crypto/rsa"
	"fmt"
	"strconv"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"golang.org/x/oauth2"
)

const (
	tokenTTL          = 15 * time.Second
	tokenExpiryBuffer = 3 * time.Second
)

// tokenSourceFunc adapts a function that returns a token into an
// [oauth2.TokenSource].
type tokenSourceFunc func() (*oauth2.Token, error)

// Token returns a new token.
func (f tokenSourceFunc) Token() (*oauth2.Token, error) {
	return f()
}

func newTokenSource(fn tokenSourceFunc) oauth2.TokenSource {
	return oauth2.ReuseTokenSourceWithExpiry(
		nil,
		fn,
		tokenExpiryBuffer,
	)
}

func generateToken(
	tokenID uint64,
	clientID string,
	privateKey *rsa.PrivateKey,
	expiresAt time.Time,
) (string, error) {
	token, err := jwt.
		NewBuilder().
		JwtID(strconv.FormatUint(tokenID, 10)).
		Issuer(clientID).
		IssuedAt(time.Now()).
		Expiration(expiresAt).
		Build()
	if err != nil {
		return "", fmt.Errorf("unable to generate github application token: %w", err)
	}

	data, err := jwt.Sign(
		token,
		jwt.WithKey(
			jwa.RS256,
			privateKey,
		),
	)
	if err != nil {
		return "", fmt.Errorf("unable to sign github application token: %w", err)
	}

	return string(data), nil
}
