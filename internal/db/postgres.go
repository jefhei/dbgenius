package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresBackend implements Database for PostgreSQL using pgx/v5.
type PostgresBackend struct {
	config PostgresConfig
	pool   *pgxpool.Pool
}

// PostgresConfig holds the connection parameters for a PostgreSQL database.
type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// NewPostgresBackend creates a new Postgres database backend.
func NewPostgresBackend(cfg PostgresConfig) *PostgresBackend {
	return &PostgresBackend{
		config: cfg,
	}
}

func (p *PostgresBackend) connString() string {
	sslMode := p.config.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		p.config.User, p.config.Password,
		p.config.Host, p.config.Port,
		p.config.DBName, sslMode,
	)
}

func (p *PostgresBackend) Connect(ctx context.Context) error {
	connStr := p.connString()
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return wrapError("postgres connect", fmt.Errorf("invalid connection string: %w", err))
	}

	// Connection pool settings
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return wrapError("postgres connect", fmt.Errorf("cannot create connection pool: %w", err))
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return wrapError("postgres connect", fmt.Errorf("cannot reach server: %w", err))
	}

	p.pool = pool
	return nil
}

func (p *PostgresBackend) Close() error {
	if p.pool != nil {
		p.pool.Close()
		p.pool = nil
	}
	return nil
}

func (p *PostgresBackend) Ping(ctx context.Context) error {
	if p.pool == nil {
		return fmt.Errorf("not connected")
	}
	return p.pool.Ping(ctx)
}

func (p *PostgresBackend) GetType() string {
	return "postgres"
}

func (p *PostgresBackend) GetSchemas(ctx context.Context) ([]string, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("not connected")
	}

	rows, err := p.pool.Query(ctx, `
		SELECT schema_name
		FROM information_schema.schemata
		WHERE schema_name NOT IN ('information_schema', 'pg_catalog', 'pg_toast')
		ORDER BY schema_name
	`)
	if err != nil {
		return nil, wrapError("list schemas", err)
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, wrapError("list schemas", err)
		}
		schemas = append(schemas, schema)
	}
	return schemas, rows.Err()
}

func (p *PostgresBackend) GetTables(ctx context.Context, schema string) ([]TableInfo, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("not connected")
	}

	rows, err := p.pool.Query(ctx, `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = $1 AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`, schema)
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

		// Get columns for this table
		columns, err := p.getColumns(ctx, schema, tableName)
		if err != nil {
			return nil, err
		}

		tables = append(tables, TableInfo{
			Schema:  schema,
			Name:    tableName,
			Columns: columns,
		})
	}
	return tables, rows.Err()
}

func (p *PostgresBackend) GetTableInfo(ctx context.Context, schema, table string) (TableInfo, error) {
	if p.pool == nil {
		return TableInfo{}, fmt.Errorf("not connected")
	}

	columns, err := p.getColumns(ctx, schema, table)
	if err != nil {
		return TableInfo{}, err
	}

	return TableInfo{
		Schema:  schema,
		Name:    table,
		Columns: columns,
	}, nil
}

func (p *PostgresBackend) getColumns(ctx context.Context, schema, table string) ([]ColumnInfo, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable,
			c.column_default,
			CASE WHEN pk.constraint_type = 'PRIMARY KEY' THEN true ELSE false END AS is_pk
		FROM information_schema.columns c
		LEFT JOIN (
			SELECT kcu.column_name, tc.constraint_type
			FROM information_schema.key_column_usage kcu
			JOIN information_schema.table_constraints tc
				ON kcu.constraint_name = tc.constraint_name
				AND kcu.table_schema = tc.table_schema
				AND kcu.table_name = tc.table_name
			WHERE tc.constraint_type = 'PRIMARY KEY'
				AND kcu.table_schema = $1
				AND kcu.table_name = $2
		) pk ON c.column_name = pk.column_name
		WHERE c.table_schema = $1 AND c.table_name = $2
		ORDER BY c.ordinal_position
	`, schema, table)
	if err != nil {
		return nil, wrapError(fmt.Sprintf("get columns for %s.%s", schema, table), err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var nullable string
		var defaultVal *string
		var isPK bool

		if err := rows.Scan(&col.Name, &col.DataType, &nullable, &defaultVal, &isPK); err != nil {
			return nil, wrapError("scan column", err)
		}
		col.Nullable = nullable == "YES"
		col.DefaultValue = defaultVal
		col.IsPrimaryKey = isPK
		columns = append(columns, col)
	}
	return columns, rows.Err()
}

func (p *PostgresBackend) ExecuteQuery(ctx context.Context, query string) (*QueryResult, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("not connected")
	}

	start := time.Now()
	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, wrapError("execute query", err)
	}
	defer rows.Close()

	// Get column names
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columns[i] = string(fd.Name)
	}

	// Fetch all rows
	var resultRows [][]*string
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, wrapError("read row", err)
		}
		row := make([]*string, len(values))
		for i, v := range values {
			if v == nil {
				row[i] = nil
			} else {
				s := fmt.Sprintf("%v", v)
				row[i] = &s
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

func (p *PostgresBackend) GetTableRowCount(ctx context.Context, schema, table string) (int64, error) {
	if p.pool == nil {
		return 0, fmt.Errorf("not connected")
	}

	var count int64
	err := p.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM %s.%s", schema, table),
	).Scan(&count)
	if err != nil {
		return 0, wrapError("get row count", err)
	}
	return count, nil
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.0fμs", float64(d.Microseconds()))
	} else if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Milliseconds()))
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
