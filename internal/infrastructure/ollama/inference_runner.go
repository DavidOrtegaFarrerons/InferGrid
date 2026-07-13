package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type InferenceRunner struct {
	client *Client
}

func NewInferenceRunner(client *Client) InferenceRunner {
	return InferenceRunner{client: client}
}

type Client struct {
	client    *http.Client
	serverUrl string
	model     string
}

func NewClient(client *http.Client, serverUrl string, model string) *Client {
	return &Client{
		client:    client,
		serverUrl: serverUrl,
		model:     model,
	}
}

const (
	generatePath = "/api/generate"
)

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(GenerateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	})

	if err != nil {
		return "", fmt.Errorf("encode generate request: %w", err)
	}

	url := strings.TrimRight(c.serverUrl, "/") + generatePath

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("create generate request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send generate request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode > http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return "", fmt.Errorf(
			"ollama returned %s: %s",
			resp.Status,
			strings.TrimSpace(string(body)),
		)
	}

	var result GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode generate response: %w", err)
	}

	if !result.Done {
		return "", fmt.Errorf("ollama generation did not complete")
	}

	return result.Response, nil
}

func (i InferenceRunner) Generate(ctx context.Context, prompt string) (string, error) {
	return i.client.Generate(ctx, prompt)
}
