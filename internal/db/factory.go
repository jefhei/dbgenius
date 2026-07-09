package db

import (
	"context"
	"fmt"
	"time"
)

// Factory creates Database backends from configuration.
type Factory struct{}

// NewFactory creates a new database factory.
func NewFactory() *Factory {
	return &Factory{}
}

// CreateBackend creates and connects a database backend with retry support.
func (f *Factory) CreateBackend(ctx context.Context, dbType string, config map[string]interface{}) (Database, error) {
	return CreateBackendWithRetry(ctx, dbType, config, DefaultRetryConfig())
}

// createBackend creates a Database backend without connecting or retry wrapping.
func (f *Factory) createBackend(ctx context.Context, dbType string, config map[string]interface{}) (Database, error) {
	switch dbType {
	case "postgres":
		cfg := PostgresConfig{
			Host:     getString(config, "host", "localhost"),
			Port:     getInt(config, "port", 5432),
			User:     getString(config, "user", "postgres"),
			Password: getString(config, "password", ""),
			DBName:   getString(config, "dbname", "postgres"),
			SSLMode:  getString(config, "sslmode", "disable"),
		}
		return NewPostgresBackend(cfg), nil

	case "sqlite":
		path := getString(config, "dbname", "data.db")
		if p, ok := config["path"]; ok {
			if s, ok := p.(string); ok {
				path = s
			}
		}
		return NewSQLiteBackend(path), nil

	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// CreateBackendWithRetry creates, connects, and wraps a database backend
// with automatic retry and reconnection logic.
func CreateBackendWithRetry(ctx context.Context, dbType string, config map[string]interface{}, retryCfg RetryConfig) (Database, error) {
	factory := &Factory{}

	// Create a context with timeout for the initial connection
	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db, err := factory.createBackend(connectCtx, dbType, config)
	if err != nil {
		return nil, err
	}

	// Wrap with retry logic
	retryDB := NewRetryDB(db, retryCfg)

	// Try connecting with retry
	if err := retryDB.ConnectWithRetry(connectCtx); err != nil {
		return nil, fmt.Errorf("cannot connect to %s: %w", dbType, FriendlyError(err))
	}

	return retryDB, nil
}

func getString(m map[string]interface{}, key, defaultVal string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return defaultVal
}

func getInt(m map[string]interface{}, key string, defaultVal int) int {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case float64:
			return int(val)
		}
	}
	return defaultVal
}
