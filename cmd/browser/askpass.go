package main

import (
	"net/http"

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

	imbue.Decorate1(
		container,
		func(
			ctx imbue.Context,
			mux *http.ServeMux,
			h *askpass.Handler,
		) (*http.ServeMux, error) {
			mux.Handle("/askpass", h)
			return mux, nil
		},
	)
}
