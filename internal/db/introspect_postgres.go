package db

import (
	"context"
	"fmt"
)

// PostgresIntrospector implements SchemaIntrospector for PostgreSQL.
type PostgresIntrospector struct {
	db Database
}

// NewPostgresIntrospector creates a new Postgres schema introspector.
func NewPostgresIntrospector(db Database) *PostgresIntrospector {
	return &PostgresIntrospector{db: db}
}

func (pi *PostgresIntrospector) Introspect(ctx context.Context) (*SchemaInfo, error) {
	schemas, err := pi.db.GetSchemas(ctx)
	if err != nil {
		return nil, fmt.Errorf("introspect: %w", err)
	}

	result := &SchemaInfo{
		Schemas: schemas,
		Tables:  make(map[string][]ExtendedTableInfo),
	}

	for _, schema := range schemas {
		tables, err := pi.IntrospectSchema(ctx, schema)
		if err != nil {
			return nil, err
		}
		result.Tables[schema] = tables
	}

	return result, nil
}

func (pi *PostgresIntrospector) IntrospectSchema(ctx context.Context, schema string) ([]ExtendedTableInfo, error) {
	tables, err := pi.db.GetTables(ctx, schema)
	if err != nil {
		return nil, fmt.Errorf("introspect schema %s: %w", schema, err)
	}

	var extended []ExtendedTableInfo
	for _, t := range tables {
		table, err := pi.IntrospectTable(ctx, schema, t.Name)
		if err != nil {
			return nil, err
		}
		extended = append(extended, *table)
	}

	return extended, nil
}

func (pi *PostgresIntrospector) IntrospectTable(ctx context.Context, schema, table string) (*ExtendedTableInfo, error) {
	tableInfo, err := pi.db.GetTableInfo(ctx, schema, table)
	if err != nil {
		return nil, fmt.Errorf("introspect table %s.%s: %w", schema, table, err)
	}

	fks, err := pi.GetForeignKeys(ctx, schema, table)
	if err != nil {
		return nil, err
	}

	indexes, err := pi.GetIndexes(ctx, schema, table)
	if err != nil {
		return nil, err
	}

	return &ExtendedTableInfo{
		Table:       tableInfo,
		ForeignKeys: fks,
		Indexes:     indexes,
	}, nil
}

func (pi *PostgresIntrospector) GetForeignKeys(ctx context.Context, schema, table string) ([]ForeignKeyInfo, error) {
	pgDb, ok := pi.db.(*PostgresBackend)
	if !ok || pgDb.pool == nil {
		return nil, nil
	}

	rows, err := pgDb.pool.Query(ctx, `
		SELECT
			kcu.column_name,
			ccu.table_schema AS ref_schema,
			ccu.table_name AS ref_table,
			ccu.column_name AS ref_column,
			tc.constraint_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON tc.constraint_name = kcu.constraint_name
			AND tc.table_schema = kcu.table_schema
			AND tc.table_name = kcu.table_name
		JOIN information_schema.constraint_column_usage ccu
			ON tc.constraint_name = ccu.constraint_name
			AND tc.table_schema = ccu.constraint_schema
		WHERE tc.constraint_type = 'FOREIGN KEY'
			AND tc.table_schema = $1
			AND tc.table_name = $2
		ORDER BY kcu.ordinal_position
	`, schema, table)
	if err != nil {
		return nil, fmt.Errorf("get foreign keys for %s.%s: %w", schema, table, err)
	}
	defer rows.Close()

	var fks []ForeignKeyInfo
	for rows.Next() {
		var fk ForeignKeyInfo
		if err := rows.Scan(&fk.ColumnName, &fk.RefSchema, &fk.RefTable, &fk.RefColumnName, &fk.ConstraintName); err != nil {
			return nil, fmt.Errorf("scan foreign key: %w", err)
		}
		fks = append(fks, fk)
	}
	return fks, rows.Err()
}

func (pi *PostgresIntrospector) GetIndexes(ctx context.Context, schema, table string) ([]IndexInfo, error) {
	pgDb, ok := pi.db.(*PostgresBackend)
	if !ok || pgDb.pool == nil {
		return nil, nil
	}

	rows, err := pgDb.pool.Query(ctx, `
		SELECT
			i.indexname AS index_name,
			i.indexdef
		FROM pg_indexes i
		WHERE i.schemaname = $1 AND i.tablename = $2
		ORDER BY i.indexname
	`, schema, table)
	if err != nil {
		return nil, fmt.Errorf("get indexes for %s.%s: %w", schema, table, err)
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var idx IndexInfo
		var indexDef string
		if err := rows.Scan(&idx.Name, &indexDef); err != nil {
			return nil, fmt.Errorf("scan index: %w", err)
		}

		// Parse columns from CREATE INDEX definition
		// Format: "CREATE [UNIQUE] INDEX name ON table (col1, col2, ...)"
		idx.IsPrimary = false // pg_indexes doesn't include PK indexes; they're managed via constraints
		idx.IsUnique = contains(indexDef, "UNIQUE")
		idx.Columns = parseIndexColumns(indexDef)

		indexes = append(indexes, idx)
	}
	return indexes, rows.Err()
}

// parseIndexColumns extracts column names from a CREATE INDEX definition.
func parseIndexColumns(indexDef string) []string {
	// Find the column list after '('
	parenStart := -1
	for i := len(indexDef) - 1; i >= 0; i-- {
		if indexDef[i] == '(' {
			parenStart = i
			break
		}
	}
	if parenStart < 0 {
		return nil
	}

	parenEnd := len(indexDef) - 1
	if indexDef[parenEnd] != ')' {
		return nil
	}

	colsStr := indexDef[parenStart+1 : parenEnd]
	if colsStr == "" {
		return nil
	}

	// Split by comma, trim whitespace
	var columns []string
	current := make([]byte, 0, len(colsStr))
	for i := 0; i < len(colsStr); i++ {
		c := colsStr[i]
		if c == ',' {
			col := string(current)
			col = trimSpace(col)
			// Handle "col DESC" or "col ASC" — take just the column name
			for j, descIdx := 0, -1; j < len(col); j++ {
				if col[j] == ' ' {
					descIdx = j
					col = col[:descIdx]
					break
				}
			}
			if col != "" {
				columns = append(columns, col)
			}
			current = current[:0]
		} else {
			current = append(current, c)
		}
	}
	if len(current) > 0 {
		col := string(current)
		col = trimSpace(col)
		// Handle "col DESC" or "col ASC"
		for j := 0; j < len(col); j++ {
			if col[j] == ' ' {
				col = col[:j]
				break
			}
		}
		if col != "" {
			columns = append(columns, col)
		}
	}

	return columns
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n') {
		end--
	}
	if start >= end {
		return ""
	}
	return s[start:end]
}
