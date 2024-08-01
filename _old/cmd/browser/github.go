package main

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/dogmatiq/browser/githubx"
	"github.com/dogmatiq/imbue"
)

func init() {
	imbue.With1(
		container,
		func(
			ctx imbue.Context,
			pk *rsa.PrivateKey,
		) (*githubx.Connector, error) {
			baseURL, _ := githubURL.Value()

			return githubx.NewConnector(
				int64(githubAppID.Value()),
				pk,
				githubAppClientID.Value(),
				githubAppClientSecret.Value(),
				baseURL,
				nil, // use default http transport
			)
		},
	)

	imbue.With0(
		container,
		func(
			ctx imbue.Context,
		) (*rsa.PrivateKey, error) {
			content := []byte(githubAppPrivateKey.Value())
			block, _ := pem.Decode(content)
			if block == nil {
				return nil, errors.New("could not load private key: no valid PEM data found")
			}

			if block.Type != "RSA PRIVATE KEY" {
				return nil, fmt.Errorf("could not load private key: expected RSA PRIVATE KEY, found %s", block.Type)
			}

			pk, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("could not load private key: %w", err)
			}

			return pk, nil
		},
	)
}
