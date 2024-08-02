package main

import (
	"log/slog"
	"net"
	"net/http"
	"time"

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

	imbue.With2(
		container,
		func(
			ctx imbue.Context,
			mux *http.ServeMux,
			logger *slog.Logger,
		) (*http.Server, error) {
			return &http.Server{
				Addr:              net.JoinHostPort("0", httpListenPort.Value()),
				Handler:           mux,
				ReadTimeout:       10 * time.Second,
				ReadHeaderTimeout: 1 * time.Second,
				WriteTimeout:      10 * time.Second,
				IdleTimeout:       30 * time.Second,
			}, nil
		},
	)
}
