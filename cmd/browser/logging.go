package main

import (
	"log/slog"

	"github.com/dogmatiq/imbue"
)

func init() {
	imbue.With0(
		container,
		func(
			ctx imbue.Context,
		) (*slog.Logger, error) {
			return slog.Default(), nil
		},
	)
}
