package db

import (
	"context"
	"fmt"
)

// ColumnInfo describes a single column in a table.
type ColumnInfo struct {
	Name         string
	DataType     string
	Nullable     bool
	DefaultValue *string
	IsPrimaryKey bool
}

// TableInfo describes a database table.
type TableInfo struct {
	Schema  string
	Name    string
	Columns []ColumnInfo
}

// QueryResult holds the result of executing a SQL query.
type QueryResult struct {
	Columns []string
	Rows    [][]*string // nil values represent SQL NULL
	RowsAffected int64
	Duration      string // human-readable duration
}

// Database defines the interface for database operations.
type Database interface {
	// Connect establishes a connection to the database.
	Connect(ctx context.Context) error

	// Close closes the database connection.
	Close() error

	// Ping checks if the connection is still alive.
	Ping(ctx context.Context) error

	// GetType returns the database type (e.g., "postgres", "sqlite").
	GetType() string

	// GetSchemas returns the list of schemas (for Postgres) or
	// returns a single entry "main" for SQLite.
	GetSchemas(ctx context.Context) ([]string, error)

	// GetTables returns all tables in a given schema.
	GetTables(ctx context.Context, schema string) ([]TableInfo, error)

	// GetTableInfo returns detailed info about a specific table.
	GetTableInfo(ctx context.Context, schema, table string) (TableInfo, error)

	// ExecuteQuery runs a SQL query and returns the results.
	ExecuteQuery(ctx context.Context, query string) (*QueryResult, error)

	// GetTableRowCount returns the number of rows in a table.
	GetTableRowCount(ctx context.Context, schema, table string) (int64, error)
}

// wrapError wraps a database error with a user-friendly message.
func wrapError(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}

// FriendlyError returns a user-readable error message.
func FriendlyError(err error) string {
	if err == nil {
		return ""
	}

	// Try to extract the underlying error message
	msg := err.Error()

	// Common error patterns
	switch {
	case contains(msg, "connection refused"):
		return "Could not connect to the database server. Is it running?"
	case contains(msg, "authentication failed") || contains(msg, "password authentication failed"):
		return "Authentication failed. Check your username and password."
	case contains(msg, "database does not exist"):
		return "The specified database does not exist on the server."
	case contains(msg, "no such host"):
		return "Could not resolve the database hostname. Check the host address."
	case contains(msg, "timeout"):
		return "Connection timed out. Check network connectivity and server status."
	case contains(msg, "SSL"):
		return "SSL connection failed. Try setting sslmode=disable."
	case contains(msg, "no such table"):
		return "The specified table does not exist in the current schema."
	case contains(msg, "syntax error"):
		return "SQL syntax error. Check your query."
	default:
		return msg
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
