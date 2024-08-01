package main

import (
	"crypto/rsa"
	"database/sql"
	"net/http"

	"github.com/dogmatiq/browser/analyzer"
	"github.com/dogmatiq/browser/githubx"
	"github.com/dogmatiq/browser/web"
	"github.com/dogmatiq/imbue"
)

func init() {
	imbue.With4(
		container,
		func(
			ctx imbue.Context,
			c *githubx.Connector,
			o *analyzer.Orchestrator,
			db *sql.DB,
			pk *rsa.PrivateKey,
		) (http.Handler, error) {
			return web.NewRouter(
				version,
				c,
				o,
				pk,
				githubAppHookSecret.Value(),
				db,
			), nil
		},
	)
}
