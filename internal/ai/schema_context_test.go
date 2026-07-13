package ai

import (
	"testing"

	"github.com/jefhei/dbgenius/internal/db"
)

func schemaInfoForTest() *db.SchemaInfo {
	return &db.SchemaInfo{
		Schemas: []string{"public"},
		Tables: map[string][]db.ExtendedTableInfo{
			"public": {
				{
					Table: db.TableInfo{
						Schema: "public",
						Name:   "employees",
						Columns: []db.ColumnInfo{
							{Name: "id", DataType: "integer", IsPrimaryKey: true, Nullable: false},
							{Name: "name", DataType: "varchar(100)", Nullable: false},
							{Name: "email", DataType: "varchar(255)", Nullable: true},
							{Name: "salary", DataType: "decimal(10,2)", Nullable: true, DefaultValue: strPtr("0.00")},
							{Name: "department_id", DataType: "integer", Nullable: true},
						},
					},
					ForeignKeys: []db.ForeignKeyInfo{
						{ColumnName: "department_id", RefSchema: "public", RefTable: "departments", RefColumnName: "id", ConstraintName: "fk_employee_dept"},
					},
					Indexes: []db.IndexInfo{
						{Name: "idx_employees_name", Columns: []string{"name"}, IsUnique: false},
						{Name: "idx_employees_email", Columns: []string{"email"}, IsUnique: true},
					},
				},
			},
		},
	}
}

func strPtr(s string) *string { return &s }

func TestNewSchemaContextBuilder(t *testing.T) {
	b := NewSchemaContextBuilder()
	if b == nil {
		t.Fatal("NewSchemaContextBuilder returned nil")
	}
}

func TestBuildContext_NilSchema(t *testing.T) {
	b := NewSchemaContextBuilder()
	ctx := b.BuildContext(nil, nil)
	if ctx != "" {
		t.Errorf("Expected empty context for nil schema, got %q", ctx)
	}
}

func TestBuildContext_WithSchema(t *testing.T) {
	b := NewSchemaContextBuilder()
	si := schemaInfoForTest()
	ctx := b.BuildContext(si, nil)

	if ctx == "" {
		t.Fatal("Expected non-empty context")
	}

	if !contains(ctx, "=== DATABASE SCHEMA ===") {
		t.Error("Context should start with header")
	}
	if !contains(ctx, "TABLE public.employees") {
		t.Error("Context should contain table name")
	}
	if !contains(ctx, "id: integer") {
		t.Error("Context should contain column info")
	}
	if !contains(ctx, "PK") {
		t.Error("Context should mark primary keys")
	}
	if !contains(ctx, "department_id -> public.departments.id") {
		t.Error("Context should show foreign keys")
	}
	if !contains(ctx, "idx_employees_name") {
		t.Error("Context should show indexes")
	}
}

func TestBuildContext_CacheHit(t *testing.T) {
	b := NewSchemaContextBuilder()
	si := schemaInfoForTest()

	ctx1 := b.BuildContext(si, nil)
	ctx2 := b.BuildContext(si, nil)

	// Same cached result
	if ctx1 != ctx2 {
		t.Error("Expected same cached context")
	}
}

func TestBuildContext_InvalidateCache(t *testing.T) {
	b := NewSchemaContextBuilder()
	si := schemaInfoForTest()

	ctx1 := b.BuildContext(si, nil)
	b.InvalidateCache()
	ctx2 := b.BuildContext(si, nil)

	// Should be equal (same data, rebuilt)
	if ctx1 != ctx2 {
		t.Error("Expected same context after rebuild from same data")
	}
}

func TestTruncatePreserveStructure_Small(t *testing.T) {
	b := NewSchemaContextBuilder()
	small := "=== DATABASE SCHEMA ===\n\nTABLE public.test\n  Columns:\n    - id: integer PK\n"
	b.maxContextLen = 5000 // larger than small
	result := b.truncatePreserveStructure(small)
	if result != small {
		t.Errorf("Expected unchanged small string, got truncated version")
	}
}

func TestTruncatePreserveStructure_Large(t *testing.T) {
	b := NewSchemaContextBuilder()
	// Build a large context that will need truncation
	var large string
	large += "=== DATABASE SCHEMA ===\n\n"
	for i := 0; i < 50; i++ {
		large += "TABLE public.table_\n"
		for j := 0; j < 20; j++ {
			large += "  - col: integer\n"
		}
	}

	b.maxContextLen = 500
	result := b.truncatePreserveStructure(large)
	if len(result) > 550 {
		t.Errorf("Truncated result too long: %d chars", len(result))
	}
	if !contains(result, "[... schema truncated due to size ...]") {
		t.Error("Truncation notice should be present")
	}
}

func TestCacheKey(t *testing.T) {
	b := NewSchemaContextBuilder()
	si := schemaInfoForTest()
	key := b.cacheKey(si)
	if key == "" {
		t.Error("Expected non-empty cache key")
	}
}

func TestBuildContext_SampleValues(t *testing.T) {
	b := NewSchemaContextBuilder()
	si := schemaInfoForTest()
	ctx := b.BuildContext(si, nil)

	if !contains(ctx, "--- Schema: public (1 tables) ---") {
		t.Error("Should show schema overview")
	}
}

// Helper for string contains
func contains(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
