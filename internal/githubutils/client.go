package githubutils

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/go-github/v63/github"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"golang.org/x/oauth2"
)

// ClientSet provides [github.Client] instances for a GitHub application and
// specific installations of that application.
type ClientSet struct {
	AppID      int64
	PrivateKey *rsa.PrivateKey
	BaseURL    *url.URL

	appClient   atomic.Pointer[github.Client]
	instClients sync.Map // map[int64]*github.Client
}

// App returns a client that is scoped to the GitHub application itself.
func (f *ClientSet) App() *github.Client {
	c := f.appClient.Load()

	if c == nil {
		c = github.NewClient(oauth2.NewClient(
			context.Background(),
			&appTokenSource{
				AppID:      f.AppID,
				PrivateKey: f.PrivateKey,
			},
		))

		if f.BaseURL != nil {
			var err error
			c, err = c.WithEnterpriseURLs(
				f.BaseURL.String(),
				f.BaseURL.String(),
			)
			if err != nil {
				panic(err)
			}
		}

		if !f.appClient.CompareAndSwap(nil, c) {
			c = f.appClient.Load()
		}
	}

	return c
}

// Installation returns a client that is scoped to the installation with the
// given ID.
func (f *ClientSet) Installation(id int64) *github.Client {
	client, ok := f.instClients.Load(id)

	if !ok {
		client, _ = f.instClients.LoadOrStore(
			id,
			github.NewClient(
				oauth2.NewClient(
					context.Background(),
					&installationTokenSource{
						AppClient:      f.App(),
						InstallationID: id,
					},
				),
			),
		)
	}

	return client.(*github.Client)
}

// InstallationTokenSource is an [oauth2.TokenSource] that generates GitHub API
// tokens scoped to a specific installation of the GitHub app.
type installationTokenSource struct {
	AppClient      *github.Client
	InstallationID int64
	Options        *github.InstallationTokenOptions
}

func (s *installationTokenSource) Token() (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
		Expiry:      token.GetExpiresAt().Time,
	}, nil
}

// appTokenSource is an implementation of oauth2.AppTokenSource that generates
// GitHub API tokens that authenticate as a GitHub application.
type appTokenSource struct {
	// AppID is the GitHub application ID.
	AppID int64

	// PrivateKey is the application's private key.
	PrivateKey *rsa.PrivateKey
}

// Token returns an OAuth token that authenticates as the GitHub application.
func (s *appTokenSource) Token() (*oauth2.Token, error) {
	expiresAt := time.Now().Add(1 * time.Minute)

	token, err := jwt.
		NewBuilder().
		Issuer(strconv.FormatInt(s.AppID, 10)).
		IssuedAt(time.Now()).
		Expiration(expiresAt).
		Build()

	if err != nil {
		return nil, fmt.Errorf("unable to build GitHub client token: %w", err)
	}

	data, err := jwt.Sign(
		token,
		jwt.WithKey(
			jwa.RS256,
			s.PrivateKey,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to sign GitHub client token: %w", err)
	}

	return &oauth2.Token{
		AccessToken: string(data),
		Expiry:      expiresAt,
	}, nil
}
