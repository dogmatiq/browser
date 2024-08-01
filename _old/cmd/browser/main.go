package main

import (
	"context"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dogmatiq/browser/analyzer"
	"github.com/dogmatiq/browser/githubx"
	"github.com/dogmatiq/dodeca/logging"
	"github.com/dogmatiq/ferrite"
	"github.com/dogmatiq/imbue"
	"github.com/google/go-github/v38/github"
)

var (
	// version is the current version, set automatically by the makefiles.
	version string

	// container is the dependency injection container.
	container = imbue.New()
)

func init() {
	rand.Seed(time.Now().UnixNano())

	imbue.With0(
		container,
		func(
			ctx imbue.Context,
		) (logging.Logger, error) {
			return logging.DebugLogger, nil
		},
	)
}

func main() {
	ferrite.Init()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	g := container.WaitGroup(ctx)

	imbue.Go2(
		g,
		func(
			ctx context.Context,
			o *analyzer.Orchestrator,
			l logging.Logger,
		) error {
			for {
				if err := o.Run(ctx); err != nil {
					if err == context.Canceled {
						return err
					}

					logging.LogString(l, err.Error())
				}
			}
		},
	)

	imbue.Go2(
		g,
		func(
			ctx context.Context,
			c *githubx.Connector,
			o *analyzer.Orchestrator,
		) error {
			return githubx.ListInstallations(
				ctx,
				c.AppClient,
				func(ctx context.Context, i *github.Installation) error {
					ic, err := c.InstallationClient(ctx, i.GetID())
					if err != nil {
						return err
					}

					return githubx.ListRepos(
						ctx,
						ic,
						func(ctx context.Context, r *github.Repository) error {
							return o.EnqueueAnalyis(ctx, r.GetID())
						},
					)
				},
			)
		},
	)

	imbue.Go1(
		g,
		func(
			ctx context.Context,
			h http.Handler,
		) error {
			s := http.Server{
				Addr:    ":8080",
				Handler: h,
			}
			go func() {
				<-ctx.Done()
				s.Close()
			}()

			return s.ListenAndServe()
		},
	)

	if err := g.Wait(); err != nil {
		panic(err)
	}
}
