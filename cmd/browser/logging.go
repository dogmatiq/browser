package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/dogmatiq/browser/components/messagelogger"
	"github.com/dogmatiq/ferrite"
	"github.com/dogmatiq/imbue"
	"github.com/dogmatiq/minibus"
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

	imbue.Decorate1(
		container,
		func(
			ctx imbue.Context,
			options []minibus.Option,
			logger *slog.Logger,
		) ([]minibus.Option, error) {
			return append(
				options,
				minibus.WithFunc(
					func(ctx context.Context) error {
						return messagelogger.Run(ctx, logger)
					},
				),
			), nil
		},
	)
}
