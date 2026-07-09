package db

import (
	"context"
	"sync"
	"time"
)

// ForeignKeyInfo describes a foreign key relationship.
type ForeignKeyInfo struct {
	ColumnName    string
	RefSchema     string
	RefTable      string
	RefColumnName string
	ConstraintName string
}

// IndexInfo describes a database index.
type IndexInfo struct {
	Name        string
	Columns     []string
	IsUnique    bool
	IsPrimary   bool
}

// ExtendedTableInfo extends TableInfo with foreign keys and indexes.
type ExtendedTableInfo struct {
	Table       TableInfo
	ForeignKeys []ForeignKeyInfo
	Indexes     []IndexInfo
}

// SchemaInfo holds complete schema metadata for a database.
type SchemaInfo struct {
	Schemas []string
	Tables  map[string][]ExtendedTableInfo // schema -> tables
}

// SchemaIntrospector provides methods for introspecting database schema metadata.
type SchemaIntrospector interface {
	// Introspect retrieves the full schema: schemas, tables, columns, FKs, indexes.
	Introspect(ctx context.Context) (*SchemaInfo, error)

	// IntrospectSchema retrieves metadata for a specific schema.
	IntrospectSchema(ctx context.Context, schema string) ([]ExtendedTableInfo, error)

	// IntrospectTable retrieves detailed metadata for a specific table.
	IntrospectTable(ctx context.Context, schema, table string) (*ExtendedTableInfo, error)

	// GetForeignKeys retrieves foreign key constraints for a table.
	GetForeignKeys(ctx context.Context, schema, table string) ([]ForeignKeyInfo, error)

	// GetIndexes retrieves indexes for a table.
	GetIndexes(ctx context.Context, schema, table string) ([]IndexInfo, error)
}

// SchemaCache provides caching for introspected schema metadata.
type SchemaCache struct {
	mu       sync.RWMutex
	schema   *SchemaInfo
	expiresAt time.Time
	ttl      time.Duration
}

// NewSchemaCache creates a new schema cache with the given TTL.
func NewSchemaCache(ttl time.Duration) *SchemaCache {
	return &SchemaCache{
		ttl: ttl,
	}
}

// Get returns the cached schema if it's still valid.
func (c *SchemaCache) Get() *SchemaInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.schema == nil || time.Now().After(c.expiresAt) {
		return nil
	}
	return c.schema
}

// Set stores a schema in the cache.
func (c *SchemaCache) Set(schema *SchemaInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.schema = schema
	c.expiresAt = time.Now().Add(c.ttl)
}

// Invalidate clears the cache.
func (c *SchemaCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.schema = nil
	c.expiresAt = time.Time{}
}

// IntrospectedBackend wraps a Database with schema introspection capabilities.
type IntrospectedBackend struct {
	Database
	introspector SchemaIntrospector
	cache        *SchemaCache
}

// NewIntrospectedBackend creates a new introspected database backend.
// It automatically uses the right introspector based on the database type.
func NewIntrospectedBackend(db Database, cacheTTL time.Duration) *IntrospectedBackend {
	var introspector SchemaIntrospector
	switch db.GetType() {
	case "postgres":
		introspector = NewPostgresIntrospector(db)
	case "sqlite":
		introspector = NewSQLiteIntrospector(db)
	}

	return &IntrospectedBackend{
		Database:     db,
		introspector: introspector,
		cache:        NewSchemaCache(cacheTTL),
	}
}

// Introspect retrieves the full schema, using cache if available.
func (ib *IntrospectedBackend) Introspect(ctx context.Context) (*SchemaInfo, error) {
	if cached := ib.cache.Get(); cached != nil {
		return cached, nil
	}

	schema, err := ib.introspector.Introspect(ctx)
	if err != nil {
		return nil, err
	}

	ib.cache.Set(schema)
	return schema, nil
}

// RefreshSchema invalidates the cache and re-introspects.
func (ib *IntrospectedBackend) RefreshSchema(ctx context.Context) (*SchemaInfo, error) {
	ib.cache.Invalidate()
	return ib.Introspect(ctx)
}

// GetIntrospector returns the underlying schema introspector.
func (ib *IntrospectedBackend) GetIntrospector() SchemaIntrospector {
	return ib.introspector
}
