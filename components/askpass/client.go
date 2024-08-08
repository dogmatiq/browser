package askpass

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
)

var httpClient = &http.Client{
	Transport: &http.Transport{
		DialContext: func(_ context.Context, _, host string) (net.Conn, error) {
			return net.Dial("unix", decodeSocketFromHost(host))
		},
	},
}

func encodeSocketAsHost(socket string) string {
	return "[" + strings.ReplaceAll(socket, "/", ":") + "]"
}

func decodeSocketFromHost(host string) string {
	host, _, _ = net.SplitHostPort(host)
	return strings.ReplaceAll(host, ":", "/")
}

// Ask sends a request for credentials to the askpass server.
func Ask(
	ctx context.Context,
	socket string,
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
		fmt.Sprintf("http://%s", encodeSocketAsHost(socket)),
		bytes.NewReader(body),
	)
	if err != nil {
		panic(err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Content-Length", strconv.Itoa(len(body)))

	httpRes, err := httpClient.Do(httpReq)
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
