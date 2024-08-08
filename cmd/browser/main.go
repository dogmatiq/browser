package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dogmatiq/ferrite"
	"github.com/dogmatiq/imbue"
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
		fmt.Fprintln(os.Stderr, err)
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

	bin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("unable to determine executable path: %w", err)
	}

	socket := bin + ".sock"

	if os.Getenv("GIT_ASKPASS") == bin {
		return runAskpass(ctx, socket)
	}

	return runServer(ctx, socket)
}
