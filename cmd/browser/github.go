package main

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dogmatiq/browser/integration/github"
	"github.com/dogmatiq/browser/internal/githubutils"
	"github.com/dogmatiq/ferrite"
	"github.com/dogmatiq/imbue"
)

var (
	githubURL = ferrite.
			URL("GITHUB_URL", "the base URL of the GitHub API").
			Optional()

	githubAppID = ferrite.
			Signed[int64]("GITHUB_APP_ID", "the ID of the GitHub application used to read repository content").
			WithMinimum(1).
			Required()

	// githubAppClientID = ferrite.
	// 			String("GITHUB_APP_CLIENT_ID", "the client ID of the GitHub application used to read repository content").
	// 			Required()

	// githubAppClientSecret = ferrite.
	// 			String("GITHUB_APP_CLIENT_SECRET", "the client secret for the GitHub application used to read repository content").
	// 			WithSensitiveContent().
	// 			Required()

	githubAppPrivateKey = ferrite.
				String("GITHUB_APP_PRIVATE_KEY", "the private key for the GitHub application used to read repository content").
				WithSensitiveContent().
				Required()

	// githubAppHookSecret = ferrite.
	// 			String("GITHUB_APP_HOOK_SECRET", "the secret used to verify GitHub web-hook requests are genuine").
	// 			WithSensitiveContent().
	// 			Required()
)

func init() {
	imbue.With2(
		container,
		func(
			ctx imbue.Context,
			clients *githubutils.ClientSet,
			logger *slog.Logger,
		) (*github.Watcher, error) {
			return &github.Watcher{
				Clients: clients,
				Logger:  logger,
			}, nil
		},
	)

	imbue.With1(
		container,
		func(
			ctx imbue.Context,
			logger *slog.Logger,
		) (*githubutils.ClientSet, error) {
			content := []byte(githubAppPrivateKey.Value())
			block, _ := pem.Decode(content)
			if block == nil {
				return nil, errors.New("could not load GitHub private key: no valid PEM data found")
			}

			if block.Type != "RSA PRIVATE KEY" {
				return nil, fmt.Errorf("could not load GitHub private key: expected RSA PRIVATE KEY, found %s", block.Type)
			}

			pk, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("could not load GitHub private key: %w", err)
			}

			baseURL, _ := githubURL.Value()

			return &githubutils.ClientSet{
				AppID:      githubAppID.Value(),
				PrivateKey: pk,
				BaseURL:    baseURL,
			}, nil
		},
	)
}
