package db

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()
	if cfg.MaxAttempts != 3 {
		t.Errorf("Default MaxAttempts = %d, want 3", cfg.MaxAttempts)
	}
	if cfg.BaseDelay != 1*time.Second {
		t.Errorf("Default BaseDelay = %v, want 1s", cfg.BaseDelay)
	}
	if cfg.MaxDelay != 10*time.Second {
		t.Errorf("Default MaxDelay = %v, want 10s", cfg.MaxDelay)
	}
	if cfg.Multiplier != 2.0 {
		t.Errorf("Default Multiplier = %f, want 2.0", cfg.Multiplier)
	}
	if cfg.JitterFactor != 0.1 {
		t.Errorf("Default JitterFactor = %f, want 0.1", cfg.JitterFactor)
	}
}

type mockDB struct {
	Database
	connectErr   error
	pingErr      error
	closeErr     error
	executeErr   error
	executeRes   *QueryResult
	connectCalls int
	pingCalls    int
	executeCalls int
	closeCalls   int
}

func (m *mockDB) Connect(ctx context.Context) error {
	m.connectCalls++
	return m.connectErr
}

func (m *mockDB) Ping(ctx context.Context) error {
	m.pingCalls++
	return m.pingErr
}

func (m *mockDB) ExecuteQuery(ctx context.Context, query string) (*QueryResult, error) {
	m.executeCalls++
	return m.executeRes, m.executeErr
}

func (m *mockDB) Close() error {
	m.closeCalls++
	return m.closeErr
}

func TestNewRetryDB(t *testing.T) {
	cfg := DefaultRetryConfig()
	mock := &mockDB{}
	rdb := NewRetryDB(mock, cfg)
	if rdb == nil {
		t.Fatal("NewRetryDB returned nil")
	}
}

func TestRetryDB_ConnectWithRetry_Success(t *testing.T) {
	cfg := DefaultRetryConfig()
	mock := &mockDB{}
	rdb := NewRetryDB(mock, cfg)

	err := rdb.ConnectWithRetry(context.Background())
	if err != nil {
		t.Errorf("ConnectWithRetry failed: %v", err)
	}
	if mock.connectCalls != 1 {
		t.Errorf("Expected 1 connect call, got %d", mock.connectCalls)
	}
}

func TestRetryDB_ConnectWithRetry_Permanent(t *testing.T) {
	cfg := DefaultRetryConfig()
	mock := &mockDB{connectErr: errors.New("authentication failed")}
	rdb := NewRetryDB(mock, cfg)

	err := rdb.ConnectWithRetry(context.Background())
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	// Permanent error should not retry
	if mock.connectCalls != 1 {
		t.Errorf("Expected 1 connect call (permanent), got %d", mock.connectCalls)
	}
}

func TestIsPermanentError_Auth(t *testing.T) {
	tests := []string{
		"authentication failed",
		"password authentication failed",
		"password is invalid",
		"permission denied",
		"no pg_hba.conf entry",
	}
	for _, msg := range tests {
		if !isPermanentError(errors.New(msg)) {
			t.Errorf("isPermanentError(%q) should be true", msg)
		}
	}
}

func TestIsPermanentError_MissingDB(t *testing.T) {
	tests := []string{
		"database does not exist",
		"database not found",
		"unknown database",
		"no such table",
	}
	for _, msg := range tests {
		if !isPermanentError(errors.New(msg)) {
			t.Errorf("isPermanentError(%q) should be true", msg)
		}
	}
}

func TestIsPermanentError_Syntax(t *testing.T) {
	tests := []string{
		"syntax error",
		"grammar error",
		"column does not exist",
	}
	for _, msg := range tests {
		if !isPermanentError(errors.New(msg)) {
			t.Errorf("isPermanentError(%q) should be true", msg)
		}
	}
}

func TestIsPermanentError_SSL(t *testing.T) {
	tests := []string{
		"SSL not supported",
		"SSL unsupported",
	}
	for _, msg := range tests {
		if !isPermanentError(errors.New(msg)) {
			t.Errorf("isPermanentError(%q) should be true", msg)
		}
	}
}

func TestIsPermanentError_Transient(t *testing.T) {
	tests := []string{
		"connection refused",
		"connection timeout",
		"network is unreachable",
		"connection reset by peer",
		"temporary failure",
	}
	for _, msg := range tests {
		if isPermanentError(errors.New(msg)) {
			t.Errorf("isPermanentError(%q) should be false (transient)", msg)
		}
	}
}

func TestIsPermanentError_Nil(t *testing.T) {
	if isPermanentError(nil) {
		t.Error("isPermanentError(nil) should be false")
	}
}

func TestRetry_ContextCancelled(t *testing.T) {
	cfg := DefaultRetryConfig()
	cfg.MaxAttempts = 5

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel

	callCount := 0
	err := retry(ctx, cfg, "test", func(ctx context.Context) error {
		callCount++
		return errors.New("transient error")
	})

	if err == nil {
		t.Fatal("Expected error from cancelled context")
	}
	if callCount < 1 {
		t.Errorf("Expected at least 1 call before cancellation, got %d", callCount)
	}
}

func TestRetry_SuccessOnFirstTry(t *testing.T) {
	cfg := DefaultRetryConfig()
	callCount := 0
	err := retry(context.Background(), cfg, "test", func(ctx context.Context) error {
		callCount++
		return nil
	})
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

func TestRetry_SuccessAfterRetry(t *testing.T) {
	cfg := DefaultRetryConfig()
	cfg.MaxAttempts = 3
	attempt := 0

	err := retry(context.Background(), cfg, "test", func(ctx context.Context) error {
		attempt++
		if attempt < 2 {
			return errors.New("transient: connection reset")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected success on retry, got %v", err)
	}
	if attempt != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempt)
	}
}

func TestRetry_Exhausted(t *testing.T) {
	cfg := DefaultRetryConfig()
	cfg.MaxAttempts = 2
	attempt := 0

	err := retry(context.Background(), cfg, "test", func(ctx context.Context) error {
		attempt++
		return errors.New("transient: timeout")
	})

	if err == nil {
		t.Fatal("Expected error after exhausting retries")
	}
	if attempt != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempt)
	}
}
