package web

import (
	"context"
	"fmt"
	"net/http"

	"github.com/dogmatiq/browser/analyzer"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v38/github"
)

func handleGitHubHook(
	version string,
	secret string,
	o *analyzer.Orchestrator,
) gin.HandlerFunc {
	secretBytes := []byte(secret)

	return func(ctx *gin.Context) {
		payload, err := github.ValidatePayload(ctx.Request, secretBytes)
		if err != nil {
			renderError(ctx, version, http.StatusForbidden)
			return
		}

		hookType := github.WebHookType(ctx.Request)
		event, err := github.ParseWebHook(hookType, payload)
		if err != nil {
			renderError(ctx, version, http.StatusBadRequest)
			return
		}

		if err := handleGitHubEvent(ctx, o, event); err != nil {
			renderError(ctx, version, http.StatusInternalServerError)
			fmt.Println("unable to handle event", err) // TODO
			return
		}

		ctx.Writer.WriteHeader(http.StatusNoContent)
	}
}

func handleGitHubEvent(
	ctx context.Context,
	o *analyzer.Orchestrator,
	event interface{},
) error {
	// TODO: handle repository rename
	switch event := event.(type) {
	case *github.InstallationEvent:
		return handleInstallationEvent(ctx, o, event)
	case *github.InstallationRepositoriesEvent:
		return handleInstallationRepositoriesEvent(ctx, o, event)
	case *github.PushEvent:
		return handlePushEvent(ctx, o, event)
	}

	return nil
}

func handleInstallationEvent(
	ctx context.Context,
	o *analyzer.Orchestrator,
	event *github.InstallationEvent,
) error {
	if event.Action == nil {
		return nil
	}

	switch *event.Action {
	case "created", "unsuspend", "new_permissions_accepted":
		for _, r := range event.Repositories {
			if err := o.EnqueueAnalyis(ctx, r.GetID()); err != nil {
				return err
			}
		}
	case "deleted", "suspend":
		for _, r := range event.Repositories {
			if err := o.EnqueueRemoval(ctx, r.GetID()); err != nil {
				return err
			}
		}
	}

	return nil
}

func handleInstallationRepositoriesEvent(
	ctx context.Context,
	o *analyzer.Orchestrator,
	event *github.InstallationRepositoriesEvent,
) error {
	for _, r := range event.RepositoriesRemoved {
		if err := o.EnqueueRemoval(ctx, r.GetID()); err != nil {
			return err
		}
	}

	for _, r := range event.RepositoriesAdded {
		if err := o.EnqueueAnalyis(ctx, r.GetID()); err != nil {
			return err
		}
	}

	return nil
}

func handlePushEvent(
	ctx context.Context,
	o *analyzer.Orchestrator,
	event *github.PushEvent,
) error {
	repo := event.GetRepo()
	defaultRef := "refs/heads/" + repo.GetDefaultBranch()

	if event.GetRef() != defaultRef {
		return nil
	}

	return o.EnqueueAnalyis(ctx, repo.GetID())
}
