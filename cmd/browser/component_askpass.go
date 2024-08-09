package main

import (
	"net"

	"github.com/dogmatiq/browser/components/askpass"
	"github.com/dogmatiq/imbue"
	"github.com/dogmatiq/minibus"
)

type askpassListener imbue.Name[net.Listener]

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

	imbue.With0Named[askpassListener](
		container,
		func(
			ctx imbue.Context,
		) (net.Listener, error) {
			lis, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				return nil, err
			}
			ctx.Defer(lis.Close)

			return lis, nil
		},
	)
}
