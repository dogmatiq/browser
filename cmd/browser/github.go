package main

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/dogmatiq/browser/components/github"
	"github.com/dogmatiq/browser/internal/githubutils"
	"github.com/dogmatiq/ferrite"
	"github.com/dogmatiq/imbue"
	"github.com/dogmatiq/minibus"
)

var (
	githubURL = ferrite.
			URL("GITHUB_URL", "the base URL of the GitHub API").
			Optional()

	githubAppClientID = ferrite.
				String("GITHUB_APP_CLIENT_ID", "the client ID of the GitHub application used to read repository content").
				Required()

	// githubAppClientSecret = ferrite.
	// 			String("GITHUB_APP_CLIENT_SECRET", "the client secret for the GitHub application used to read repository content").
	// 			WithSensitiveContent().
	// 			Required()

	githubAppPrivateKey = ferrite.
				String("GITHUB_APP_PRIVATE_KEY", "the private key for the GitHub application used to read repository content").
				WithSensitiveContent().
				Required()

	githubAppHookSecret = ferrite.
				String("GITHUB_APP_HOOK_SECRET", "the secret used to verify GitHub web-hook requests are genuine").
				WithSensitiveContent().
				Required()
)

func init() {
	imbue.With1(
		container,
		func(
			ctx imbue.Context,
			clients *githubutils.ClientSet,
		) (*github.CredentialServer, error) {
			return &github.CredentialServer{
				Clients: clients,
			}, nil
		},
	)

	imbue.With2(
		container,
		func(
			ctx imbue.Context,
			clients *githubutils.ClientSet,
			logger *slog.Logger,
		) (*github.RepositoryWatcher, error) {
			return &github.RepositoryWatcher{
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
		) (*github.WebHookHandler, error) {
			return &github.WebHookHandler{
				Secret: githubAppHookSecret.Value(),
				Logger: logger,
			}, nil
		},
	)

	imbue.Decorate3(
		container,
		func(
			ctx imbue.Context,
			options []minibus.Option,
			s *github.CredentialServer,
			w *github.RepositoryWatcher,
			h *github.WebHookHandler,
		) ([]minibus.Option, error) {
			return append(
				options,
				minibus.WithFunc(s.Run),
				minibus.WithFunc(w.Run),
				minibus.WithFunc(h.Run),
			), nil
		},
	)

	imbue.Decorate1(
		container,
		func(
			ctx imbue.Context,
			mux *http.ServeMux,
			h *github.WebHookHandler,
		) (*http.ServeMux, error) {
			mux.Handle("/github/hook", h)
			return mux, nil
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
				ClientID:   githubAppClientID.Value(),
				PrivateKey: pk,
				BaseURL:    baseURL,
			}, nil
		},
	)
}
