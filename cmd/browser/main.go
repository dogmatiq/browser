package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dogmatiq/browser/integration/github"
	"github.com/dogmatiq/ferrite"
	"github.com/dogmatiq/imbue"
	"github.com/dogmatiq/minibus"
)

var (
	// version is the current version, set automatically by the makefiles.
	version string

	// container is the dependency injection container.
	container = imbue.New()
)

func main() {
	ferrite.Init()

	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	return imbue.Invoke1(
		ctx,
		container,
		func(
			ctx context.Context,
			gh *github.Watcher,
		) error {
			return minibus.Run(
				ctx,
				minibus.WithFunc(gh.Run),
			)
		},
	)
}
