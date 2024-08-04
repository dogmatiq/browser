package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dogmatiq/ferrite"
	"github.com/dogmatiq/imbue"
	"github.com/dogmatiq/minibus"
	"golang.org/x/sync/errgroup"
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

	g := container.WaitGroup(ctx)

	imbue.Go1(
		g,
		func(
			ctx context.Context,
			options []minibus.Option,
		) error {
			return minibus.Run(ctx, options...)
		},
	)

	imbue.Go2(
		g,
		func(
			ctx context.Context,
			server *http.Server,
			logger *slog.Logger,
		) error {
			g, ctx := errgroup.WithContext(ctx)

			server.BaseContext = func(l net.Listener) context.Context {
				logger.InfoContext(
					ctx,
					"listening for HTTP requests",
					slog.String("listen_address", l.Addr().String()),
				)
				return ctx
			}

			g.Go(func() error {
				<-ctx.Done()

				shutdownCtx := context.WithoutCancel(ctx)
				shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()

				return server.Shutdown(shutdownCtx)
			})

			g.Go(func() error {
				return server.ListenAndServe()
			})

			return g.Wait()
		},
	)

	return g.Wait()
}
