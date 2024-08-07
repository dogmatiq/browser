package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/dogmatiq/browser/messages"
	"github.com/dogmatiq/ferrite"
	"github.com/go-git/go-git/v5"
	"github.com/google/uuid"
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

	req := messages.RepoCredentialsRequest{
		CorrelationID: uuid.New(),
		RepoURL:       remote.Config().URLs[0],
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("unable to marshal request: %w", err)
	}

	httpRes, err := http.Post(
		fmt.Sprintf("http://127.0.0.1:%s/askpass", httpListenPort.Value()),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("unable to POST to /askpass: %w", err)
	}
	defer httpRes.Body.Close()

	if httpRes.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", httpRes.StatusCode)
	}

	body, err = io.ReadAll(httpRes.Body)
	if err != nil {
		return fmt.Errorf("unable to read response body: %w", err)
	}

	var res messages.RepoCredentialsResponse

	if err := json.Unmarshal(body, &res); err != nil {
		return fmt.Errorf("unable to unmarshal response: %w", err)
	}

	switch {
	case strings.HasPrefix(os.Args[1], "Username "):
		fmt.Println(res.Username)
	case strings.HasPrefix(os.Args[1], "Password "):
		fmt.Println(res.Password)
	default:
		return fmt.Errorf("unexpected prompt: %s", os.Args[1])
	}

	return nil
}
