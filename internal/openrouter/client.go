package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"diaryhero/internal/config"
)

const chatCompletionsPath = "/chat/completions"

type Client struct {
	baseURL       string
	apiKey        string
	primaryModel  string
	fallbackModel string
	siteURL       string
	appName       string
	httpClient    *http.Client
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model    string    `json:"model,omitempty"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream,omitempty"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func NewClient(cfg config.OpenRouterConfig) *Client {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")

	return &Client{
		baseURL:       baseURL,
		apiKey:        cfg.APIKey,
		primaryModel:  cfg.PrimaryModel,
		fallbackModel: cfg.FallbackModel,
		siteURL:       cfg.SiteURL,
		appName:       cfg.AppName,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *Client) Enabled() bool {
	return c.apiKey != ""
}

func (c *Client) ChatCompletion(ctx context.Context, request ChatCompletionRequest) (ChatCompletionResponse, error) {
	if !c.Enabled() {
		return ChatCompletionResponse{}, fmt.Errorf("openrouter api key is not configured")
	}

	if request.Model == "" {
		request.Model = c.primaryModel
	}

	response, err := c.doChatCompletion(ctx, request)
	if err == nil || c.fallbackModel == "" || c.fallbackModel == request.Model {
		return response, err
	}

	request.Model = c.fallbackModel
	return c.doChatCompletion(ctx, request)
}

func (c *Client) doChatCompletion(ctx context.Context, request ChatCompletionRequest) (ChatCompletionResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("marshal request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+chatCompletionsPath, bytes.NewReader(body))
	if err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("create request: %w", err)
	}

	httpRequest.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpRequest.Header.Set("Content-Type", "application/json")
	if c.siteURL != "" {
		httpRequest.Header.Set("HTTP-Referer", c.siteURL)
	}
	if c.appName != "" {
		httpRequest.Header.Set("X-Title", c.appName)
	}

	httpResponse, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("send request: %w", err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode >= http.StatusBadRequest {
		var errorResponse ErrorResponse
		if decodeErr := json.NewDecoder(httpResponse.Body).Decode(&errorResponse); decodeErr == nil && errorResponse.Error.Message != "" {
			return ChatCompletionResponse{}, fmt.Errorf("openrouter returned %s: %s", httpResponse.Status, errorResponse.Error.Message)
		}
		return ChatCompletionResponse{}, fmt.Errorf("openrouter returned %s", httpResponse.Status)
	}

	var response ChatCompletionResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&response); err != nil {
		return ChatCompletionResponse{}, fmt.Errorf("decode response: %w", err)
	}

	return response, nil
}

func (c *Client) Timeout() time.Duration {
	if c.httpClient == nil {
		return 0
	}

	return c.httpClient.Timeout
}
