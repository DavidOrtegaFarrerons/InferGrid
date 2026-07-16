package openai_compatible

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
	apiKey string
}

func NewClient(client *http.Client, serverUrl string, model string, apiKey string) *Client {
	return &Client{
		client:    client,
		serverUrl: serverUrl,
		model:     model,
		apiKey: apiKey,
	}
}

const (
	generatePath = "/v1/chat/completions"
)

type GenerateRequest struct {
	Model    string                    `json:"model"`
	Messages []GenerateRequestMessages `json:"messages"`
	Stream   bool                      `json:"stream"`
}

type GenerateRequestMessages struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GenerateResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(GenerateRequest{
		Model: c.model,
		Messages: []GenerateRequestMessages{
			{Role: "user", Content: prompt},
		},
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
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send generate request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode > http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return "", fmt.Errorf(
			"openai compatible returned %s: %s",
			resp.Status,
			strings.TrimSpace(string(body)),
		)
	}

	var result GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode generate response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("openai compatible response had no choices")
	}

	return result.Choices[0].Message.Content, nil
}

func (i InferenceRunner) Generate(ctx context.Context, prompt string) (string, error) {
	return i.client.Generate(ctx, prompt)
}
