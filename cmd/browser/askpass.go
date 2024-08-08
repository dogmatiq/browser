package main

import (
	"log/slog"

	"github.com/dogmatiq/browser/components/askpass"
	"github.com/dogmatiq/imbue"
	"github.com/dogmatiq/minibus"
)

func init() {
	imbue.With1(
		container,
		func(
			ctx imbue.Context,
			logger *slog.Logger,
		) (*askpass.Handler, error) {
			// Note this handler ISNT added to the [http.ServeMux], it's served
			// only via a unix socket.
			return &askpass.Handler{
				Logger: logger,
			}, nil
		},
	)

	imbue.Decorate1(
		container,
		func(
			ctx imbue.Context,
			options []minibus.Option,
			h *askpass.Handler,
		) ([]minibus.Option, error) {
			return append(options, minibus.WithFunc(h.Run)), nil
		},
	)
}
