package askpass

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/dogmatiq/browser/model"
	"github.com/google/uuid"
)

// Ask sends a request for credentials to the askpass server.
func Ask(
	ctx context.Context,
	addr string,
	requestID uuid.UUID,
	repoURL string,
	cred model.CredentialType,
) (string, error) {
	req := apiRequest{
		ID:         requestID,
		URL:        repoURL,
		Credential: cred,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("unable to marshal askpass request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("http://%s", addr),
		bytes.NewReader(body),
	)
	if err != nil {
		panic(err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Content-Length", strconv.Itoa(len(body)))

	// retry:
	httpRes, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("unable to send askpass request: %w", err)
	}
	defer httpRes.Body.Close()

	body, err = io.ReadAll(httpRes.Body)
	if err != nil {
		return "", fmt.Errorf("unable to read askpass response: %w", err)
	}

	if httpRes.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"unable to perform askpass exchange: [%d] %s",
			httpRes.StatusCode,
			string(body),
		)
	}

	var res apiResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return "", fmt.Errorf("unable to unmarshal askpass response: %w", err)
	}

	return res.Value, nil
}

// apiRequest is a apiRequest for credentials sent over the HTTP API.
type apiRequest struct {
	ID         uuid.UUID            `json:"id"`
	URL        string               `json:"url"`
	Credential model.CredentialType `json:"credential"`
}

// apiResponse is the apiResponse to an [apiRequest].
type apiResponse struct {
	Value string `json:"value"`
}
