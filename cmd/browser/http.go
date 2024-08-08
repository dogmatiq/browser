package main

import (
	"net/http"

	"github.com/dogmatiq/ferrite"
	"github.com/dogmatiq/imbue"
)

var (
	httpListenPort = ferrite.
		NetworkPort("HTTP_LISTEN_PORT", "the port to listen on for HTTP requests").
		WithDefault("8080").
		Required()
)

func init() {
	imbue.With0(
		container,
		func(
			ctx imbue.Context,
		) (*http.ServeMux, error) {
			return http.NewServeMux(), nil
		},
	)
}
