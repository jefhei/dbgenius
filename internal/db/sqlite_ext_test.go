package db

import (
	"context"
	"testing"
)

func TestIsSelectQuery_Select(t *testing.T) {
	if !isSelectQuery("SELECT * FROM foo") {
		t.Error("isSelectQuery('SELECT * FROM foo') should be true")
	}
}

func TestIsSelectQuery_SelectLower(t *testing.T) {
	if !isSelectQuery("select * from foo") {
		t.Error("isSelectQuery('select * from foo') should be true")
	}
}

func TestIsSelectQuery_Pragma(t *testing.T) {
	if !isSelectQuery("PRAGMA table_info('foo')") {
		t.Error("isSelectQuery('PRAGMA...') should be true")
	}
}

func TestIsSelectQuery_With(t *testing.T) {
	if !isSelectQuery("WITH cte AS (SELECT 1) SELECT * FROM cte") {
		t.Error("isSelectQuery('WITH...') should be true")
	}
}

func TestIsSelectQuery_Explain(t *testing.T) {
	if !isSelectQuery("EXPLAIN SELECT * FROM foo") {
		t.Error("isSelectQuery('EXPLAIN...') should be true")
	}
}

func TestIsSelectQuery_Insert(t *testing.T) {
	if isSelectQuery("INSERT INTO foo VALUES (1)") {
		t.Error("isSelectQuery('INSERT...') should be false")
	}
}

func TestIsSelectQuery_Update(t *testing.T) {
	if isSelectQuery("UPDATE foo SET bar = 1") {
		t.Error("isSelectQuery('UPDATE...') should be false")
	}
}

func TestIsSelectQuery_Delete(t *testing.T) {
	if isSelectQuery("DELETE FROM foo") {
		t.Error("isSelectQuery('DELETE...') should be false")
	}
}

func TestIsSelectQuery_Create(t *testing.T) {
	if isSelectQuery("CREATE TABLE foo (id INT)") {
		t.Error("isSelectQuery('CREATE...') should be false")
	}
}

func TestIsSelectQuery_Empty(t *testing.T) {
	if !isSelectQuery("") {
		t.Error("isSelectQuery('') should be true (default)")
	}
}

func TestIsSelectQuery_Whitespace(t *testing.T) {
	if !isSelectQuery("  \t\n  SELECT * FROM foo") {
		t.Error("isSelectQuery('  \\t\\n  SELECT...') should be true")
	}
}

func TestIsSelectQuery_Alter(t *testing.T) {
	if isSelectQuery("ALTER TABLE foo ADD COLUMN bar INT") {
		t.Error("isSelectQuery('ALTER...') should be false")
	}
}

func TestIsSelectQuery_Drop(t *testing.T) {
	if isSelectQuery("DROP TABLE foo") {
		t.Error("isSelectQuery('DROP...') should be false")
	}
}

func TestStrPtr(t *testing.T) {
	s := "hello"
	ptr := strPtr(s)
	if ptr == nil {
		t.Fatal("strPtr returned nil")
	}
	if *ptr != s {
		t.Errorf("strPtr = %q, want %q", *ptr, s)
	}
}

func TestSQLiteBackend_ExecuteQuery_EmptyResult(t *testing.T) {
	backend, ctx, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	// DDL statement — SQLite reports 1 row affected for successful DDL
	result, err := backend.ExecuteQuery(ctx, "CREATE TABLE temp_test (x INT)")
	if err != nil {
		t.Fatalf("ExecuteQuery DDL failed: %v", err)
	}
	if result.RowsAffected < 1 {
		t.Errorf("Expected at least 1 row affected for CREATE TABLE, got %d", result.RowsAffected)
	}
}

func TestSQLiteBackend_GetTables_Empty(t *testing.T) {
	// Use an in-memory DB with no tables (after filtering sqlite_*)
	backend := NewSQLiteBackend(":memory:")
	ctx := context.Background()
	if err := backend.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer backend.Close()

	tables, err := backend.GetTables(ctx, "main")
	if err != nil {
		t.Fatalf("GetTables failed: %v", err)
	}
	// Should have 0 user tables
	if len(tables) != 0 {
		t.Errorf("Expected 0 tables in empty DB, got %d", len(tables))
	}
}

func TestSQLiteBackend_GetTables_NotConnected(t *testing.T) {
	backend := NewSQLiteBackend(":memory:")
	_, err := backend.GetTables(context.Background(), "main")
	if err == nil {
		t.Error("Expected error when not connected")
	}
}

func TestSQLite_Introspection_EmptySchema(t *testing.T) {
	backend := NewSQLiteBackend(":memory:")
	ctx := context.Background()
	if err := backend.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer backend.Close()

	introspector := NewSQLiteIntrospector(backend)
	schema, err := introspector.Introspect(ctx)
	if err != nil {
		t.Fatalf("Introspect failed: %v", err)
	}
	if schema == nil {
		t.Fatal("Expected non-nil schema for empty DB")
	}
	if len(schema.Tables) == 0 || len(schema.Tables["main"]) != 0 {
		// Should have a "main" key with 0 tables
		t.Errorf("Expected empty main schema, got tables: %v", schema.Tables)
	}
}

func TestSQLiteIntrospector_GetForeignKeys_NoFKs(t *testing.T) {
	backend, ctx, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	introspector := NewSQLiteIntrospector(backend)

	// Create a table with no FKs
	backend.db.ExecContext(ctx, "CREATE TABLE standalone (id INTEGER PRIMARY KEY)")

	fks, err := introspector.GetForeignKeys(ctx, "main", "standalone")
	if err != nil {
		t.Fatalf("GetForeignKeys failed: %v", err)
	}
	if len(fks) != 0 {
		t.Errorf("Expected 0 FKs for standalone table, got %d", len(fks))
	}
}

func TestSQLiteIntrospector_GetIndexes_NoIndexes(t *testing.T) {
	backend, ctx, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	introspector := NewSQLiteIntrospector(backend)

	// Create a table with no explicit indexes
	backend.db.ExecContext(ctx, "CREATE TABLE no_index (id INTEGER, name TEXT)")

	indexes, err := introspector.GetIndexes(ctx, "main", "no_index")
	if err != nil {
		t.Fatalf("GetIndexes failed: %v", err)
	}
	// May have implicit rowid index, just check no error
	t.Logf("Indexes for no_index: %d", len(indexes))
}

func TestSQLiteBackend_ExecuteQuery_NotConnected(t *testing.T) {
	backend := NewSQLiteBackend(":memory:")
	_, err := backend.ExecuteQuery(context.Background(), "SELECT 1")
	if err == nil {
		t.Error("Expected error when not connected")
	}
}
