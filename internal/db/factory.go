package db

import (
	"context"
	"fmt"
)

// Factory creates a Database backend based on configuration.
type Factory struct{}

// NewFactory creates a new database factory.
func NewFactory() *Factory {
	return &Factory{}
}

// CreateBackend creates and connects a database backend from config.
// configType is "postgres", "sqlite", or "mysql"
// config is a map of connection parameters
func (f *Factory) CreateBackend(ctx context.Context, dbType string, config map[string]interface{}) (Database, error) {
	var backend Database

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
		backend = NewPostgresBackend(cfg)

	case "sqlite":
		path := getString(config, "path", config["dbname"].(string))
		if path == "" {
			path = getString(config, "dbname", "data.db")
		}
		backend = NewSQLiteBackend(path)

	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	if err := backend.Connect(ctx); err != nil {
		return nil, fmt.Errorf("cannot connect to %s: %w", dbType, err)
	}

	return backend, nil
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
