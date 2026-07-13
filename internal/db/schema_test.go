package db

import (
	"testing"
	"time"
)

func TestSchemaCache_GetSet(t *testing.T) {
	cache := NewSchemaCache(1 * time.Minute)

	// Initially nil
	if got := cache.Get(); got != nil {
		t.Error("Expected nil for empty cache")
	}

	schema := &SchemaInfo{
		Schemas: []string{"main"},
		Tables:  map[string][]ExtendedTableInfo{},
	}
	cache.Set(schema)

	if got := cache.Get(); got == nil {
		t.Fatal("Expected non-nil after Set")
	} else if len(got.Schemas) != 1 || got.Schemas[0] != "main" {
		t.Errorf("Got schemas %v, want [main]", got.Schemas)
	}
}

func TestSchemaCache_Expired(t *testing.T) {
	cache := NewSchemaCache(0) // TTL of 0 means immediately expired

	schema := &SchemaInfo{
		Schemas: []string{"main"},
		Tables:  map[string][]ExtendedTableInfo{},
	}
	cache.Set(schema)

	// With TTL=0, cache should be immediately expired
	time.Sleep(1 * time.Millisecond)
	if got := cache.Get(); got != nil {
		t.Error("Expected nil for expired cache")
	}
}

func TestSchemaCache_Invalidate(t *testing.T) {
	cache := NewSchemaCache(1 * time.Minute)

	schema := &SchemaInfo{Schemas: []string{"main"}, Tables: map[string][]ExtendedTableInfo{}}
	cache.Set(schema)
	cache.Invalidate()

	if got := cache.Get(); got != nil {
		t.Error("Expected nil after Invalidate")
	}
}

func TestNewIntrospectedBackend_SQLite(t *testing.T) {
	db := NewSQLiteBackend(":memory:")
	ib := NewIntrospectedBackend(db, 5*time.Minute)

	if ib == nil {
		t.Fatal("NewIntrospectedBackend returned nil")
	}
	if ib.GetType() != "sqlite" {
		t.Errorf("Expected sqlite, got %s", ib.GetType())
	}
	if ib.GetIntrospector() == nil {
		t.Error("Expected non-nil introspector")
	}
}

func TestNewIntrospectedBackend_Postgres(t *testing.T) {
	db := NewSQLiteBackend(":memory:")
	ib := NewIntrospectedBackend(db, 5*time.Minute)

	if ib.GetType() != "sqlite" {
		t.Errorf("Expected sqlite, got %s", ib.GetType())
	}

	introspector := ib.GetIntrospector()
	if _, ok := introspector.(*SQLiteIntrospector); !ok {
		t.Errorf("Expected SQLiteIntrospector, got %T", introspector)
	}
}

func TestNewIntrospectedBackend_UnknownType(t *testing.T) {
	db := NewSQLiteBackend(":memory:")
	ib := NewIntrospectedBackend(db, 5*time.Minute)
	// SQLite backend gets a SQLiteIntrospector
	introspector := ib.GetIntrospector()
	if introspector == nil {
		t.Error("Expected a SQLiteIntrospector for sqlite type")
	}
}

func TestSchemaCache_GetReturnsNilForEmpty(t *testing.T) {
	cache := NewSchemaCache(5 * time.Minute)
	if got := cache.Get(); got != nil {
		t.Error("Empty cache should return nil")
	}
}

func TestIntrospectedBackend_CacheNotExpired(t *testing.T) {
	// Test that cache returns the same value within TTL
	cache := NewSchemaCache(1 * time.Hour)
	schema := &SchemaInfo{
		Schemas: []string{"public"},
		Tables:  map[string][]ExtendedTableInfo{},
	}
	cache.Set(schema)

	// Should get cached result
	if got := cache.Get(); got == nil {
		t.Error("Expected cached schema")
	} else if len(got.Schemas) != 1 || got.Schemas[0] != "public" {
		t.Errorf("Got schemas %v, want [public]", got.Schemas)
	}
}

func TestIntrospectedBackend_RefreshAfterInvalidate(t *testing.T) {
	// Test cache invalidation
	cache := NewSchemaCache(1 * time.Hour)
	schema1 := &SchemaInfo{
		Schemas: []string{"public"},
		Tables:  map[string][]ExtendedTableInfo{},
	}
	cache.Set(schema1)

	schema1Cached := cache.Get()
	if schema1Cached == nil {
		t.Fatal("Expected cached schema")
	}

	cache.Invalidate()
	if got := cache.Get(); got != nil {
		t.Error("Expected nil after invalidation")
	}

	// Set again
	cache.Set(&SchemaInfo{Schemas: []string{"new"}, Tables: map[string][]ExtendedTableInfo{}})
	schema2 := cache.Get()
	if schema2 == nil {
		t.Fatal("Expected non-nil after re-set")
	}
	if len(schema2.Schemas) != 1 || schema2.Schemas[0] != "new" {
		t.Errorf("Got schemas %v, want [new]", schema2.Schemas)
	}
}
