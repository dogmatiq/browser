package main

import (
	"log/slog"
	"os"

	"github.com/dogmatiq/ferrite"
	"github.com/dogmatiq/imbue"
)

var (
	debugEnabled = ferrite.
		Bool("DEBUG", "enable debug logging").
		WithDefault(false).
		Required()
)

func init() {
	imbue.With0(
		container,
		func(
			ctx imbue.Context,
		) (*slog.Logger, error) {
			level := slog.LevelInfo
			if debugEnabled.Value() {
				level = slog.LevelDebug
			}

			return slog.New(
				slog.NewTextHandler(
					os.Stderr,
					&slog.HandlerOptions{
						Level: level,
					},
				),
			), nil
		},
	)
}
