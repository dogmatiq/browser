package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dogmatiq/browser/components/askpass"
	"github.com/go-git/go-git/v5"
)

func runAskpass(ctx context.Context) error {
	if len(os.Args) < 2 {
		return fmt.Errorf("expected at least one argument")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	repo, err := git.PlainOpen(".")
	if err != nil {
		return fmt.Errorf("unable to open repository: %w", err)
	}

	remote, err := repo.Remote("origin")
	if err != nil {
		return fmt.Errorf("unable to get origin: %w", err)
	}

	username, password, err := askpass.Ask(
		ctx,
		os.Getenv("ASKPASS_ADDR"),
		remote.Config().URLs[0],
	)
	if err != nil {
		return fmt.Errorf("unable to ask for credentials: %w", err)
	}

	switch {
	case strings.HasPrefix(os.Args[1], "Username "):
		fmt.Println(username)
	case strings.HasPrefix(os.Args[1], "Password "):
		fmt.Println(password)
	default:
		return fmt.Errorf("unexpected prompt: %s", os.Args[1])
	}

	return nil
}
