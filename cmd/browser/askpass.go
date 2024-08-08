package main

import (
	"github.com/dogmatiq/browser/components/askpass"
	"github.com/dogmatiq/imbue"
	"github.com/dogmatiq/minibus"
)

func init() {
	imbue.With0(
		container,
		func(
			ctx imbue.Context,
		) (*askpass.Handler, error) {
			// Note this handler ISNT added to the [http.ServeMux], it's served
			// only via a unix socket.
			return &askpass.Handler{}, nil
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
