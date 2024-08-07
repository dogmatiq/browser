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
				minibus.WithInboxSize(10000),
			}, nil
		},
	)
}
