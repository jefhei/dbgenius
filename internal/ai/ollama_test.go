package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c := NewClient("http://localhost:11434", "llama3.2", 30*time.Second)
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.BaseURL() != "http://localhost:11434" {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL(), "http://localhost:11434")
	}
	if c.DefaultModel() != "llama3.2" {
		t.Errorf("DefaultModel = %q, want %q", c.DefaultModel(), "llama3.2")
	}
}

func TestNewClient_DefaultTimeout(t *testing.T) {
	c := NewClient("http://localhost:11434", "llama3.2", 0)
	if c.httpClient.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", c.httpClient.Timeout)
	}
}

func TestClient_HealthCheck_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("Expected /api/tags, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tagsResponse{
			Models: []ModelInfo{
				{Name: "llama3.2:latest", Size: 4000000000, ModifiedAt: "2024-01-01"},
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "llama3.2", 5*time.Second)
	err := c.HealthCheck(context.Background())
	if err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}

func TestClient_HealthCheck_ConnectionRefused(t *testing.T) {
	// Use a port that's unlikely to have a server
	c := NewClient("http://localhost:1", "llama3.2", 1*time.Second)
	err := c.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("Expected error for connection refused")
	}
	if _, ok := err.(ConnectionError); !ok {
		t.Errorf("Expected ConnectionError, got %T: %v", err, err)
	}
}

func TestClient_ListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("Expected /api/tags, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tagsResponse{
			Models: []ModelInfo{
				{Name: "llama3.2:latest", Size: 4000000000},
				{Name: "mistral:latest", Size: 4000000000},
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "llama3.2", 5*time.Second)
	models, err := c.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}
	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}
}

func TestClient_IsModelAvailable_Found(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(tagsResponse{
			Models: []ModelInfo{
				{Name: "llama3.2:latest", Size: 4000000000},
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "llama3.2", 5*time.Second)
	available, err := c.IsModelAvailable(context.Background(), "llama3.2")
	if err != nil {
		t.Fatalf("IsModelAvailable failed: %v", err)
	}
	if !available {
		t.Error("Expected model to be available")
	}
}

func TestClient_IsModelAvailable_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(tagsResponse{
			Models: []ModelInfo{
				{Name: "llama3.2:latest", Size: 4000000000},
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "llama3.2", 5*time.Second)
	available, err := c.IsModelAvailable(context.Background(), "nonexistent-model")
	if err != nil {
		t.Fatalf("IsModelAvailable failed: %v", err)
	}
	if available {
		t.Error("Expected nonexistent model to be unavailable")
	}
}

func TestClient_IsModelAvailable_UsesDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(tagsResponse{
			Models: []ModelInfo{
				{Name: "llama3.2:latest", Size: 4000000000},
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "llama3.2", 5*time.Second)
	available, err := c.IsModelAvailable(context.Background(), "") // Uses default
	if err != nil {
		t.Fatalf("IsModelAvailable failed: %v", err)
	}
	if !available {
		t.Error("Expected default model to be available")
	}
}

func TestClient_EnsureModelAvailable_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(tagsResponse{
			Models: []ModelInfo{
				{Name: "llama3.2:latest", Size: 4000000000},
			},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "llama3.2", 5*time.Second)
	err := c.EnsureModelAvailable(context.Background(), "llama3.2")
	if err != nil {
		t.Errorf("EnsureModelAvailable failed: %v", err)
	}
}

func TestClient_EnsureModelAvailable_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(tagsResponse{
			Models: []ModelInfo{},
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "llama3.2", 5*time.Second)
	err := c.EnsureModelAvailable(context.Background(), "nonexistent-model")
	if err == nil {
		t.Fatal("Expected ModelNotFoundError, got nil")
	}
	if _, ok := err.(ModelNotFoundError); !ok {
		t.Errorf("Expected ModelNotFoundError, got %T: %v", err, err)
	}
}

func TestClient_Generate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("Expected /api/generate, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		var req GenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}
		if req.Stream {
			t.Error("Expected non-streaming request")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GenerateResponse{
			Model:    "llama3.2",
			Response: "This is the response.",
			Done:     true,
		})
	}))
	defer server.Close()

	c := NewClient(server.URL, "llama3.2", 5*time.Second)
	resp, err := c.Generate(context.Background(), "llama3.2", "test prompt")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp != "This is the response." {
		t.Errorf("Response = %q, want %q", resp, "This is the response.")
	}
}

func TestClient_Generate_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	c := NewClient(server.URL, "llama3.2", 5*time.Second)
	_, err := c.Generate(context.Background(), "llama3.2", "test")
	if err == nil {
		t.Fatal("Expected error for 500 response")
	}
	if _, ok := err.(ServerError); !ok {
		t.Errorf("Expected ServerError, got %T: %v", err, err)
	}
}

func TestClient_GenerateStream_Success(t *testing.T) {
	tokens := []string{"Hello", " ", "World", "!"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("Expected /api/generate, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("Expected http.Flusher")
		}
		for i, token := range tokens {
			chunk := struct {
				Model    string `json:"model"`
				Response string `json:"response"`
				Done     bool   `json:"done"`
			}{
				Model:    "llama3.2",
				Response: token,
				Done:     i == len(tokens)-1,
			}
			data, _ := json.Marshal(chunk)
			w.Write(data)
			w.Write([]byte("\n"))
			flusher.Flush()
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, "llama3.2", 5*time.Second)

	var receivedTokens []string
	var finalResponse string
	callback := func(token string, done bool, fullResponse string) {
		receivedTokens = append(receivedTokens, token)
		if done {
			finalResponse = fullResponse
		}
	}

	err := c.GenerateStream(context.Background(), "llama3.2", "test prompt", callback)
	if err != nil {
		t.Fatalf("GenerateStream failed: %v", err)
	}

	if len(receivedTokens) != len(tokens) {
		t.Errorf("Expected %d tokens, got %d: %v", len(tokens), len(receivedTokens), receivedTokens)
	}
	if finalResponse != "Hello World!" {
		t.Errorf("Full response = %q, want %q", finalResponse, "Hello World!")
	}
}

func TestClient_GenerateStream_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	c := NewClient(server.URL, "llama3.2", 5*time.Second)
	err := c.GenerateStream(context.Background(), "llama3.2", "test", func(s string, b bool, s2 string) {})
	if err == nil {
		t.Fatal("Expected error for 500")
	}
}

func TestClient_GenerateStream_StreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("Expected http.Flusher")
		}
		// Send a token then an error chunk
		chunk1 := `{"model":"llama3.2","response":"Hello","done":false}`
		w.Write([]byte(chunk1 + "\n"))
		flusher.Flush()

		errorChunk := `{"error":"model not loaded"}`
		w.Write([]byte(errorChunk + "\n"))
		flusher.Flush()
	}))
	defer server.Close()

	c := NewClient(server.URL, "llama3.2", 5*time.Second)
	err := c.GenerateStream(context.Background(), "llama3.2", "test", func(s string, b bool, s2 string) {})
	if err == nil {
		t.Fatal("Expected error for stream error chunk")
	}
	if !strings.Contains(err.Error(), "model not loaded") {
		t.Errorf("Error = %v, want 'model not loaded'", err)
	}
}

func TestConnectionError_Error(t *testing.T) {
	err := ConnectionError{URL: "http://localhost:11434", Inner: nil}
	msg := err.Error()
	if !strings.Contains(msg, "cannot connect to Ollama at http://localhost:11434") {
		t.Errorf("ConnectionError message = %q", msg)
	}
}

func TestConnectionError_ErrorWithInner(t *testing.T) {
	inner := context.DeadlineExceeded
	err := ConnectionError{URL: "http://localhost:11434", Inner: inner}
	msg := err.Error()
	if !strings.Contains(msg, "context deadline exceeded") {
		t.Errorf("ConnectionError with inner = %q", msg)
	}
}

func TestModelNotFoundError_Error(t *testing.T) {
	err := ModelNotFoundError{Model: "llama3.2"}
	msg := err.Error()
	want := `model "llama3.2" is not available in Ollama`
	if msg != want {
		t.Errorf("ModelNotFoundError = %q, want %q", msg, want)
	}
}

func TestServerError_Error(t *testing.T) {
	err := ServerError{Status: 500, Message: "internal server error"}
	msg := err.Error()
	want := "Ollama server error (500): internal server error"
	if msg != want {
		t.Errorf("ServerError = %q, want %q", msg, want)
	}
}

func TestServerError_ErrorNoMessage(t *testing.T) {
	err := ServerError{Status: 404}
	msg := err.Error()
	want := "Ollama server returned HTTP 404"
	if msg != want {
		t.Errorf("ServerError = %q, want %q", msg, want)
	}
}

func TestFriendlyError_Nil(t *testing.T) {
	if got := FriendlyError(nil); got != "" {
		t.Errorf("FriendlyError(nil) = %q, want empty", got)
	}
}

func TestFriendlyError_ConnectionError(t *testing.T) {
	err := ConnectionError{URL: "http://localhost:11434"}
	got := FriendlyError(err)
	if !strings.Contains(got, "Ollama is not reachable") {
		t.Errorf("FriendlyError(ConnectionError) = %q", got)
	}
}

func TestFriendlyError_ModelNotFound(t *testing.T) {
	err := ModelNotFoundError{Model: "llama3.2"}
	got := FriendlyError(err)
	if !strings.Contains(got, "Model") || !strings.Contains(got, "llama3.2") {
		t.Errorf("FriendlyError(ModelNotFound) = %q", got)
	}
}

func TestFriendlyError_ServerError(t *testing.T) {
	err := ServerError{Status: 500, Message: "oom"}
	got := FriendlyError(err)
	if !strings.Contains(got, "Ollama server error") {
		t.Errorf("FriendlyError(ServerError) = %q", got)
	}
}

func TestFriendlyError_Generic(t *testing.T) {
	err := context.DeadlineExceeded
	got := FriendlyError(err)
	if !strings.Contains(got, "Ollama error") {
		t.Errorf("FriendlyError(generic) = %q", got)
	}
}
