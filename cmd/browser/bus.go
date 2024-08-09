package main

import (
	"github.com/dogmatiq/imbue"
	"github.com/dogmatiq/minibus"
)

func init() {
	imbue.With0(
		container,
		func(ctx imbue.Context) ([]minibus.Option, error) {
			return []minibus.Option{
				// minibus.WithBusSize(100),
				minibus.WithInboxSize(100),
			}, nil
		},
	)
}
