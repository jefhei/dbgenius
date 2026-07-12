package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Error types for Ollama client operations.
type (
	// ConnectionError indicates the Ollama server could not be reached.
	ConnectionError struct {
		URL   string
		Inner error
	}

	// ModelNotFoundError indicates the requested model is not available.
	ModelNotFoundError struct {
		Model string
	}

	// ServerError indicates the Ollama server returned a non-200 status.
	ServerError struct {
		Status  int
		Message string
	}
)

func (e ConnectionError) Error() string {
	if e.Inner != nil {
		return fmt.Sprintf("cannot connect to Ollama at %s: %v", e.URL, e.Inner)
	}
	return fmt.Sprintf("cannot connect to Ollama at %s", e.URL)
}

func (e ModelNotFoundError) Error() string {
	return fmt.Sprintf("model %q is not available in Ollama", e.Model)
}

func (e ServerError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("Ollama server error (%d): %s", e.Status, e.Message)
	}
	return fmt.Sprintf("Ollama server returned HTTP %d", e.Status)
}

// ModelInfo represents an Ollama model from the /api/tags response.
type ModelInfo struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modified_at"`
}

// tagsResponse maps the GET /api/tags response body.
type tagsResponse struct {
	Models []ModelInfo `json:"models"`
}

// Client provides an HTTP client for the Ollama API.
type Client struct {
	baseURL    string
	defaultModel string
	httpClient *http.Client
}

// NewClient creates a new Ollama API client.
func NewClient(baseURL, defaultModel string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		defaultModel: defaultModel,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// DefaultModel returns the configured default model name.
func (c *Client) DefaultModel() string {
	return c.defaultModel
}

// BaseURL returns the configured Ollama server URL.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// HealthCheck verifies the Ollama server is reachable by calling GET /api/tags.
// Returns nil if reachable, or a ConnectionError on failure.
func (c *Client) HealthCheck(ctx context.Context) error {
	_, err := c.listModels(ctx)
	if err != nil {
		return ConnectionError{
			URL:   c.baseURL,
			Inner: err,
		}
	}
	return nil
}

// ListModels returns all available models from the Ollama server.
func (c *Client) ListModels(ctx context.Context) ([]ModelInfo, error) {
	return c.listModels(ctx)
}

// IsModelAvailable checks whether a specific model is available in Ollama.
func (c *Client) IsModelAvailable(ctx context.Context, modelName string) (bool, error) {
	if modelName == "" {
		modelName = c.defaultModel
	}

	models, err := c.ListModels(ctx)
	if err != nil {
		return false, err
	}

	for _, m := range models {
		// Ollama model names can include tags (e.g., "llama3.2:latest")
		if m.Name == modelName || strings.HasPrefix(m.Name, modelName+":") {
			return true, nil
		}
	}

	return false, nil
}

// EnsureModelAvailable checks if a model exists and returns an error if not.
func (c *Client) EnsureModelAvailable(ctx context.Context, modelName string) error {
	if modelName == "" {
		modelName = c.defaultModel
	}

	available, err := c.IsModelAvailable(ctx, modelName)
	if err != nil {
		return err
	}

	if !available {
		return ModelNotFoundError{Model: modelName}
	}

	return nil
}

// listModels performs the actual GET /api/tags call.
func (c *Client) listModels(ctx context.Context) ([]ModelInfo, error) {
	url := c.baseURL + "/api/tags"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, wrapConnectionError(c.baseURL, err)
	}
	defer resp.Body.Close()

	// Read and check response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, ServerError{
			Status:  resp.StatusCode,
			Message: strings.TrimSpace(string(body)),
		}
	}

	var result tagsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Models, nil
}

// GenerateRequest is the request body for POST /api/generate.
// Used by later tasks (M3.5-M3.7) — defined here for shared use.
type GenerateRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Stream  bool   `json:"stream"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// GenerateResponse is the response body from POST /api/generate (non-streaming).
type GenerateResponse struct {
	Model     string `json:"model"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	EvalCount int    `json:"eval_count,omitempty"`
	EvalDur   int64  `json:"eval_duration,omitempty"`
}

// Generate sends a prompt to Ollama and returns the full response (non-streaming).
func (c *Client) Generate(ctx context.Context, model, prompt string) (string, error) {
	if model == "" {
		model = c.defaultModel
	}

	url := c.baseURL + "/api/generate"

	reqBody := GenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", wrapConnectionError(c.baseURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", ServerError{
			Status:  resp.StatusCode,
			Message: strings.TrimSpace(string(respBody)),
		}
	}

	var result GenerateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Response, nil
}

// wrapConnectionError wraps an HTTP error into a ConnectionError suitable for
// user-friendly display.
func wrapConnectionError(baseURL string, err error) error {
	return ConnectionError{
		URL:   baseURL,
		Inner: err,
	}
}

// FriendlyError returns a user-readable error message for Ollama errors.
func FriendlyError(err error) string {
	if err == nil {
		return ""
	}

	switch e := err.(type) {
	case ConnectionError:
		return fmt.Sprintf("⚠️  Ollama is not reachable at %s. Is it running?", e.URL)
	case ModelNotFoundError:
		return fmt.Sprintf("⚠️  Model %q not found. Pull it with: ollama pull %s", e.Model, e.Model)
	case ServerError:
		return fmt.Sprintf("⚠️  Ollama server error: %s", e.Message)
	default:
		return fmt.Sprintf("⚠️  Ollama error: %s", err.Error())
	}
}
