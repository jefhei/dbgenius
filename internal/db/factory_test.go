package db

import (
	"testing"
)

func TestGetString(t *testing.T) {
	m := map[string]interface{}{
		"host": "localhost",
		"port": 5432,
	}

	tests := []struct {
		key        string
		defaultVal string
		want       string
	}{
		{"host", "default", "localhost"},
		{"port", "default", "default"}, // int, not string
		{"missing", "fallback", "fallback"},
		{"empty", "fallback", "fallback"},
	}

	for _, tt := range tests {
		m["empty"] = ""
		got := getString(m, tt.key, tt.defaultVal)
		if got != tt.want {
			t.Errorf("getString(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestGetInt(t *testing.T) {
	m := map[string]interface{}{
		"port":    5432,
		"float":   float64(3306),
		"string":  "bad",
	}

	tests := []struct {
		key        string
		defaultVal int
		want       int
	}{
		{"port", 0, 5432},
		{"float", 0, 3306},
		{"missing", 999, 999},
	}

	for _, tt := range tests {
		got := getInt(m, tt.key, tt.defaultVal)
		if got != tt.want {
			t.Errorf("getInt(%q) = %d, want %d", tt.key, got, tt.want)
		}
	}
}

func TestFactory_CreateBackend_UnsupportedType(t *testing.T) {
	f := NewFactory()
	_, err := f.createBackend(nil, "mysql", nil)
	if err == nil {
		t.Fatal("Expected error for unsupported type, got nil")
	}
}

func TestFactory_CreateBackend_SQLite(t *testing.T) {
	f := NewFactory()
	db, err := f.createBackend(nil, "sqlite", map[string]interface{}{
		"dbname": ":memory:",
	})
	if err != nil {
		t.Fatalf("Failed to create SQLite backend: %v", err)
	}
	if db.GetType() != "sqlite" {
		t.Errorf("Expected sqlite, got %s", db.GetType())
	}
}

func TestFactory_CreateBackend_Postgres(t *testing.T) {
	f := NewFactory()
	db, err := f.createBackend(nil, "postgres", map[string]interface{}{
		"host":   "localhost",
		"port":   5432,
		"user":   "test",
		"dbname": "testdb",
	})
	if err != nil {
		t.Fatalf("Failed to create Postgres backend: %v", err)
	}
	if db.GetType() != "postgres" {
		t.Errorf("Expected postgres, got %s", db.GetType())
	}
}

func TestFactory_CreateBackend_SQLiteUsesPath(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]interface{}
		wantPath string
	}{
		{
			name: "path takes precedence",
			config: map[string]interface{}{
				"path":   "/tmp/test.db",
				"dbname": "fallback.db",
			},
			wantPath: "/tmp/test.db",
		},
		{
			name: "fallback to dbname",
			config: map[string]interface{}{
				"dbname": "data.db",
			},
			wantPath: "data.db",
		},
		{
			name:     "default path",
			config:   map[string]interface{}{},
			wantPath: "data.db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			be, err := (&Factory{}).createBackend(nil, "sqlite", tt.config)
			if err != nil {
				t.Fatalf("createBackend failed: %v", err)
			}
			sb, ok := be.(*SQLiteBackend)
			if !ok {
				t.Fatal("Expected *SQLiteBackend")
			}
			if sb.path != tt.wantPath {
				t.Errorf("path = %q, want %q", sb.path, tt.wantPath)
			}
		})
	}
}
