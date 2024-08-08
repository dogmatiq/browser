package askpass

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// Ask sends a request for credentials to the askpass server.
func Ask(
	ctx context.Context,
	port string,
	repoURL string,
) (username, password string, err error) {
	req := request{
		RepoURL: repoURL,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", "", fmt.Errorf("unable to marshal askpass request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("http://127.0.0.1:%s/askpass", port),
		bytes.NewReader(body),
	)
	if err != nil {
		panic(err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Content-Length", strconv.Itoa(len(body)))

	httpRes, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", "", fmt.Errorf("unable to send askpass request: %w", err)
	}
	defer httpRes.Body.Close()

	body, err = io.ReadAll(httpRes.Body)
	if err != nil {
		return "", "", fmt.Errorf("unable to read askpass response: %w", err)
	}

	if httpRes.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf(
			"unable to perform askpass exchange: [%d] %s",
			httpRes.StatusCode,
			string(body),
		)
	}

	var res response
	if err := json.Unmarshal(body, &res); err != nil {
		return "", "", fmt.Errorf("unable to unmarshal askpass response: %w", err)
	}

	return res.Username, res.Password, nil
}

// request is a request for credentials sent over the HTTP API.
type request struct {
	RepoURL string `json:"repo_url"`
}

// response is the response to an [request].
type response struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
