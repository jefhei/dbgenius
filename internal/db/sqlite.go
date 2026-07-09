package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteBackend implements Database for SQLite using modernc.org/sqlite (pure Go, no CGO).
type SQLiteBackend struct {
	path   string
	db     *sql.DB
}

// NewSQLiteBackend creates a new SQLite database backend.
func NewSQLiteBackend(path string) *SQLiteBackend {
	return &SQLiteBackend{
		path: path,
	}
}

func (s *SQLiteBackend) Connect(ctx context.Context) error {
	db, err := sql.Open("sqlite", s.path)
	if err != nil {
		return wrapError("sqlite connect", fmt.Errorf("cannot open database: %w", err))
	}

	// Connection pool settings
	db.SetMaxOpenConns(1) // SQLite only supports one writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return wrapError("sqlite connect", fmt.Errorf("cannot reach database: %w", err))
	}

	s.db = db
	return nil
}

func (s *SQLiteBackend) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *SQLiteBackend) Ping(ctx context.Context) error {
	if s.db == nil {
		return fmt.Errorf("not connected")
	}
	return s.db.PingContext(ctx)
}

func (s *SQLiteBackend) GetType() string {
	return "sqlite"
}

func (s *SQLiteBackend) GetSchemas(ctx context.Context) ([]string, error) {
	// SQLite doesn't have schemas like Postgres; return "main" as default
	return []string{"main"}, nil
}

func (s *SQLiteBackend) GetTables(ctx context.Context, schema string) ([]TableInfo, error) {
	if s.db == nil {
		return nil, fmt.Errorf("not connected")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT name FROM sqlite_master
		WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
		ORDER BY name
	`)
	if err != nil {
		return nil, wrapError("list tables", err)
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, wrapError("list tables", err)
		}

		columns, err := s.getColumns(ctx, tableName)
		if err != nil {
			return nil, err
		}

		tables = append(tables, TableInfo{
			Schema:  "main",
			Name:    tableName,
			Columns: columns,
		})
	}
	return tables, rows.Err()
}

func (s *SQLiteBackend) GetTableInfo(ctx context.Context, schema, table string) (TableInfo, error) {
	if s.db == nil {
		return TableInfo{}, fmt.Errorf("not connected")
	}

	columns, err := s.getColumns(ctx, table)
	if err != nil {
		return TableInfo{}, err
	}

	return TableInfo{
		Schema:  schema,
		Name:    table,
		Columns: columns,
	}, nil
}

func (s *SQLiteBackend) getColumns(ctx context.Context, table string) ([]ColumnInfo, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info('%s')", table))
	if err != nil {
		return nil, wrapError(fmt.Sprintf("get columns for %s", table), err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var cid int
		var col ColumnInfo
		var nullable int
		var defaultVal *string
		var pk int

		if err := rows.Scan(&cid, &col.Name, &col.DataType, &nullable, &defaultVal, &pk); err != nil {
			return nil, wrapError("scan column", err)
		}
		col.Nullable = nullable == 0
		col.DefaultValue = defaultVal
		col.IsPrimaryKey = pk > 0
		columns = append(columns, col)
	}
	return columns, rows.Err()
}

func (s *SQLiteBackend) ExecuteQuery(ctx context.Context, query string) (*QueryResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("not connected")
	}

	start := time.Now()

	// Determine if this is a SELECT-like query or a mutation
	isSelect := isSelectQuery(query)

	if isSelect {
		rows, err := s.db.QueryContext(ctx, query)
		if err != nil {
			return nil, wrapError("execute query", err)
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			return nil, wrapError("get columns", err)
		}

		var resultRows [][]*string
		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				return nil, wrapError("read row", err)
			}

			row := make([]*string, len(columns))
			for i, v := range values {
				if v == nil {
					row[i] = nil
				} else {
					// Convert []byte to string for proper display
					switch val := v.(type) {
					case []byte:
						s := string(val)
						row[i] = &s
					default:
						s := fmt.Sprintf("%v", val)
						row[i] = &s
					}
				}
			}
			resultRows = append(resultRows, row)
		}

		if err := rows.Err(); err != nil {
			return nil, wrapError("read rows", err)
		}

		duration := time.Since(start)
		return &QueryResult{
			Columns: columns,
			Rows:    resultRows,
			Duration: formatDuration(duration),
		}, nil
	}

	// Mutation query (INSERT, UPDATE, DELETE, CREATE, etc.)
	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return nil, wrapError("execute query", err)
	}

	rowsAffected, _ := result.RowsAffected()
	duration := time.Since(start)
	return &QueryResult{
		Columns:      []string{"result"},
		Rows:         [][]*string{{strPtr("Query executed successfully")}},
		RowsAffected: rowsAffected,
		Duration:     formatDuration(duration),
	}, nil
}

func (s *SQLiteBackend) GetTableRowCount(ctx context.Context, schema, table string) (int64, error) {
	if s.db == nil {
		return 0, fmt.Errorf("not connected")
	}

	var count int64
	err := s.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM \"%s\"", table),
	).Scan(&count)
	if err != nil {
		return 0, wrapError("get row count", err)
	}
	return count, nil
}

func isSelectQuery(query string) bool {
	// Simple heuristic: check if the trimmed query starts with SELECT or PRAGMA
	// We trim leading whitespace but keep it simple
	for i := 0; i < len(query); i++ {
		c := query[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			continue
		}
		// Check first non-whitespace character
		upper := c
		if upper >= 'a' && upper <= 'z' {
			upper -= 32
		}
		// Check if it starts with S (SELECT), P (PRAGMA, PRAGMA), W (WITH), or E (EXPLAIN)
		return upper == 'S' || upper == 'P' || upper == 'W' || upper == 'E'
	}
	return true // Default to SELECT for empty queries
}

func strPtr(s string) *string {
	return &s
}
