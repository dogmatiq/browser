package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/dogmatiq/browser/components/askpass"
	"github.com/dogmatiq/imbue"
	"github.com/dogmatiq/minibus"
)

func runServer(ctx context.Context, socket string) error {
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
			handler *http.ServeMux,
			logger *slog.Logger,
		) error {
			lis, err := net.Listen("tcp", net.JoinHostPort("0", httpListenPort.Value()))
			if err != nil {
				return err
			}
			defer lis.Close()

			logger.InfoContext(
				ctx,
				"listening for web requests",
				slog.String("address", lis.Addr().String()),
			)

			server := &http.Server{
				Handler:           handler,
				ReadTimeout:       10 * time.Second,
				ReadHeaderTimeout: 1 * time.Second,
				WriteTimeout:      10 * time.Second,
				IdleTimeout:       30 * time.Second,
			}

			go shutdownWhenDone(ctx, server)

			return server.Serve(lis)
		},
	)

	imbue.Go2(
		g,
		func(
			ctx context.Context,
			handler *askpass.Handler,
			logger *slog.Logger,
		) error {
			defer os.Remove(socket)

			lis, err := net.Listen("unix", socket)
			if err != nil {
				return err
			}
			defer lis.Close()

			logger.InfoContext(
				ctx,
				"listening for askpass requests",
				slog.String("address", lis.Addr().String()),
			)

			server := &http.Server{
				Handler:           handler,
				ReadTimeout:       10 * time.Second,
				ReadHeaderTimeout: 1 * time.Second,
				WriteTimeout:      10 * time.Second,
				IdleTimeout:       30 * time.Second,
				BaseContext: func(l net.Listener) context.Context {
					return ctx
				},
			}

			go shutdownWhenDone(ctx, server)

			return server.Serve(lis)
		},
	)

	return g.Wait()
}

func shutdownWhenDone(ctx context.Context, server *http.Server) {
	<-ctx.Done()

	shutdownCtx := context.WithoutCancel(ctx)
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	server.Shutdown(shutdownCtx)
	server.Close()
}
