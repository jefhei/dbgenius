package db

import (
	"context"
	"database/sql"
	"fmt"
)

// SQLiteIntrospector implements SchemaIntrospector for SQLite.
type SQLiteIntrospector struct {
	db Database
}

// NewSQLiteIntrospector creates a new SQLite schema introspector.
func NewSQLiteIntrospector(db Database) *SQLiteIntrospector {
	return &SQLiteIntrospector{db: db}
}

func (si *SQLiteIntrospector) Introspect(ctx context.Context) (*SchemaInfo, error) {
	schemas, err := si.db.GetSchemas(ctx)
	if err != nil {
		return nil, fmt.Errorf("introspect: %w", err)
	}

	result := &SchemaInfo{
		Schemas: schemas,
		Tables:  make(map[string][]ExtendedTableInfo),
	}

	for _, schema := range schemas {
		tables, err := si.IntrospectSchema(ctx, schema)
		if err != nil {
			return nil, err
		}
		result.Tables[schema] = tables
	}

	return result, nil
}

func (si *SQLiteIntrospector) IntrospectSchema(ctx context.Context, schema string) ([]ExtendedTableInfo, error) {
	tables, err := si.db.GetTables(ctx, schema)
	if err != nil {
		return nil, fmt.Errorf("introspect schema %s: %w", schema, err)
	}

	var extended []ExtendedTableInfo
	for _, t := range tables {
		table, err := si.IntrospectTable(ctx, schema, t.Name)
		if err != nil {
			return nil, err
		}
		extended = append(extended, *table)
	}

	return extended, nil
}

func (si *SQLiteIntrospector) IntrospectTable(ctx context.Context, schema, table string) (*ExtendedTableInfo, error) {
	tableInfo, err := si.db.GetTableInfo(ctx, schema, table)
	if err != nil {
		return nil, fmt.Errorf("introspect table %s: %w", table, err)
	}

	fks, err := si.GetForeignKeys(ctx, schema, table)
	if err != nil {
		return nil, err
	}

	indexes, err := si.GetIndexes(ctx, schema, table)
	if err != nil {
		return nil, err
	}

	return &ExtendedTableInfo{
		Table:       tableInfo,
		ForeignKeys: fks,
		Indexes:     indexes,
	}, nil
}

func (si *SQLiteIntrospector) GetForeignKeys(ctx context.Context, schema, table string) ([]ForeignKeyInfo, error) {
	sqliteDb, ok := si.db.(*SQLiteBackend)
	if !ok || sqliteDb.db == nil {
		return nil, nil
	}

	rows, err := sqliteDb.db.QueryContext(ctx, fmt.Sprintf("PRAGMA foreign_key_list('%s')", table))
	if err != nil {
		return nil, fmt.Errorf("get foreign keys for %s: %w", table, err)
	}
	defer rows.Close()

	var fks []ForeignKeyInfo
	for rows.Next() {
		var id int
		var seq int
		var fk ForeignKeyInfo
		var onUpdate, onDelete string
		var match string

		if err := rows.Scan(&id, &seq, &fk.RefTable, &fk.ColumnName, &fk.RefColumnName, &onUpdate, &onDelete, &match); err != nil {
			return nil, fmt.Errorf("scan foreign key: %w", err)
		}
		fk.RefSchema = "main"
		fk.ConstraintName = fmt.Sprintf("fk_%s_%d", table, id)
		fks = append(fks, fk)
	}
	return fks, rows.Err()
}

func (si *SQLiteIntrospector) GetIndexes(ctx context.Context, schema, table string) ([]IndexInfo, error) {
	sqliteDb, ok := si.db.(*SQLiteBackend)
	if !ok || sqliteDb.db == nil {
		return nil, nil
	}

	// Get index list
	rows, err := sqliteDb.db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_list('%s')", table))
	if err != nil {
		return nil, fmt.Errorf("get index list for %s: %w", table, err)
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin string
		var partial int

		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, fmt.Errorf("scan index: %w", err)
		}

		// Get columns for this index
		idxInfo, err := si.getIndexColumns(ctx, sqliteDb.db, name)
		if err != nil {
			return nil, err
		}

		indexes = append(indexes, IndexInfo{
			Name:      name,
			Columns:   idxInfo,
			IsUnique:  unique == 1,
			IsPrimary: origin == "pk",
		})
	}
	return indexes, rows.Err()
}

func (si *SQLiteIntrospector) getIndexColumns(ctx context.Context, db *sql.DB, indexName string) ([]string, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA index_info('%s')", indexName))
	if err != nil {
		return nil, fmt.Errorf("get index info for %s: %w", indexName, err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var seqno int
		var cid int
		var name string

		if err := rows.Scan(&seqno, &cid, &name); err != nil {
			return nil, fmt.Errorf("scan index column: %w", err)
		}
		columns = append(columns, name)
	}
	return columns, rows.Err()
}
