package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/dogmatiq/browser/components/askpass"
	"github.com/dogmatiq/imbue"
	"github.com/dogmatiq/minibus"
)

func runServer(ctx context.Context) error {
	g := container.WaitGroup(ctx)

	imbue.Go1(
		g,
		func(
			ctx context.Context,
			funcs []minibus.Func,
		) error {
			return minibus.Run(ctx, funcs...)
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
				WriteTimeout:      30 * time.Second,
				IdleTimeout:       60 * time.Second,
			}

			go shutdownWhenDone(ctx, server)

			return server.Serve(lis)
		},
	)

	imbue.Go3(
		g,
		func(
			ctx context.Context,
			handler *askpass.Handler,
			lis imbue.ByName[askpassListener, net.Listener],
			logger *slog.Logger,
		) error {
			logger.InfoContext(
				ctx,
				"listening for askpass requests",
				slog.String("address", lis.Value().Addr().String()),
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

			return server.Serve(lis.Value())
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
