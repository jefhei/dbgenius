package db

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

// RetryConfig holds configuration for retry behavior.
type RetryConfig struct {
	MaxAttempts   int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	Multiplier    float64
	JitterFactor  float64
}

// DefaultRetryConfig returns sensible defaults for retry behavior.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		BaseDelay:    1 * time.Second,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
		JitterFactor: 0.1,
	}
}

// RetryDB wraps a Database with retry logic for transient errors.
type RetryDB struct {
	Database
	config RetryConfig
}

// NewRetryDB creates a new database wrapper with automatic retry.
func NewRetryDB(db Database, config RetryConfig) *RetryDB {
	return &RetryDB{
		Database: db,
		config:   config,
	}
}

// ConnectWithRetry connects to the database with retry on failure.
func (r *RetryDB) ConnectWithRetry(ctx context.Context) error {
	return retry(ctx, r.config, "connect", func(ctx context.Context) error {
		return r.Database.Connect(ctx)
	})
}

// ExecuteQuery executes a query with retry on connection errors.
func (r *RetryDB) ExecuteQuery(ctx context.Context, query string) (*QueryResult, error) {
	var result *QueryResult
	err := retry(ctx, r.config, "execute query", func(ctx context.Context) error {
		// Try to ping first to detect stale connections
		if pingErr := r.Database.Ping(ctx); pingErr != nil {
			// Connection is stale — try to reconnect
			if reconnectErr := r.Database.Connect(ctx); reconnectErr != nil {
				return fmt.Errorf("reconnect failed: %w", reconnectErr)
			}
		}

		var queryErr error
		result, queryErr = r.Database.ExecuteQuery(ctx, query)
		return queryErr
	})
	return result, err
}

// PingWithRetry checks connectivity with retry.
func (r *RetryDB) PingWithRetry(ctx context.Context) error {
	return retry(ctx, r.config, "ping", func(ctx context.Context) error {
		return r.Database.Ping(ctx)
	})
}

// retry executes the given function with exponential backoff.
// It only retries on transient errors (connection, network, timeout).
// Permanent errors (auth failure, bad query, missing table) are not retried.
func retry(ctx context.Context, cfg RetryConfig, operation string, fn func(context.Context) error) error {
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		err := fn(ctx)
		if err == nil {
			return nil
		}
		lastErr = err

		// Check if this is a permanent error that shouldn't be retried
		if isPermanentError(err) {
			return err
		}

		// If this was the last attempt, return the error
		if attempt == cfg.MaxAttempts {
			return fmt.Errorf("%s failed after %d attempts: %w", operation, cfg.MaxAttempts, err)
		}

		// Calculate delay with exponential backoff and jitter
		delay := time.Duration(float64(cfg.BaseDelay) * math.Pow(cfg.Multiplier, float64(attempt-1)))
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}

		// Add jitter (±jitterFactor%)
		jitterRange := time.Duration(float64(delay) * cfg.JitterFactor)
		if jitterRange > 0 {
			jitter := time.Duration(rand.Int63n(int64(jitterRange) * 2)) - jitterRange
			delay += jitter
			if delay < 0 {
				delay = 0
			}
		}

		// Wait for the delay or context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s cancelled: %w", operation, ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return lastErr
}

// isPermanentError checks if the error should not be retried.
func isPermanentError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	// Authentication/authorization errors — not transient
	if contains(msg, "authentication failed") ||
		contains(msg, "auth") && contains(msg, "fail") ||
		contains(msg, "password") && contains(msg, "invalid") ||
		contains(msg, "permission denied") ||
		contains(msg, "no pg_hba.conf entry") {
		return true
	}

	// Missing resources — not transient
	if contains(msg, "database") && (contains(msg, "does not exist") || contains(msg, "not found")) ||
		contains(msg, "unknown database") ||
		contains(msg, "no such table") ||
		contains(msg, "no such host") && !contains(msg, "connection refused") {
		return true
	}

	// Syntax errors — not transient
	if contains(msg, "syntax error") ||
		contains(msg, "grammar") ||
		contains(msg, "syntax") && contains(msg, "error") ||
		contains(msg, "column") && contains(msg, "does not exist") {
		return true
	}

	// SSL configuration errors — not transient
	if contains(msg, "ssl") && (contains(msg, "not supported") || contains(msg, "unsupported")) {
		return true
	}

	return false
}
