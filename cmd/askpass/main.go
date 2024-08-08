package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/dogmatiq/browser/components/askpass"
	"github.com/dogmatiq/ferrite"
	"github.com/go-git/go-git/v5"
)

var (
	httpListenPort = ferrite.
		NetworkPort("HTTP_LISTEN_PORT", "the port to listen on for HTTP requests").
		Required()
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

	if len(os.Args) < 2 {
		return fmt.Errorf("expected at least one argument")
	}

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
		httpListenPort.Value(),
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
