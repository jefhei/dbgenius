package db

import (
	"testing"
)

func TestParseIndexColumns_Normal(t *testing.T) {
	def := `CREATE INDEX idx_name ON employees (last_name, first_name)`
	cols := parseIndexColumns(def)
	expected := []string{"last_name", "first_name"}
	if len(cols) != len(expected) {
		t.Fatalf("Expected %d columns, got %d: %v", len(expected), len(cols), cols)
	}
	for i, col := range cols {
		if col != expected[i] {
			t.Errorf("Column[%d] = %q, want %q", i, col, expected[i])
		}
	}
}

func TestParseIndexColumns_Unique(t *testing.T) {
	def := `CREATE UNIQUE INDEX idx_email ON employees (email)`
	cols := parseIndexColumns(def)
	if len(cols) != 1 || cols[0] != "email" {
		t.Errorf("Expected [email], got %v", cols)
	}
}

func TestParseIndexColumns_WithDesc(t *testing.T) {
	def := `CREATE INDEX idx_date ON orders (created_at DESC)`
	cols := parseIndexColumns(def)
	if len(cols) != 1 || cols[0] != "created_at" {
		t.Errorf("Expected [created_at], got %v", cols)
	}
}

func TestParseIndexColumns_WithAsc(t *testing.T) {
	def := `CREATE INDEX idx_name ON items (name ASC)`
	cols := parseIndexColumns(def)
	if len(cols) != 1 || cols[0] != "name" {
		t.Errorf("Expected [name], got %v", cols)
	}
}

func TestParseIndexColumns_Single(t *testing.T) {
	def := `CREATE INDEX idx_single ON t (id)`
	cols := parseIndexColumns(def)
	if len(cols) != 1 || cols[0] != "id" {
		t.Errorf("Expected [id], got %v", cols)
	}
}

func TestParseIndexColumns_NoParen(t *testing.T) {
	def := `no parentheses here`
	cols := parseIndexColumns(def)
	if cols != nil {
		t.Errorf("Expected nil, got %v", cols)
	}
}

func TestParseIndexColumns_NoClosingParen(t *testing.T) {
	def := `CREATE INDEX idx ON t (id`
	cols := parseIndexColumns(def)
	if cols != nil {
		t.Errorf("Expected nil for unclosed paren, got %v", cols)
	}
}

func TestParseIndexColumns_EmptyParen(t *testing.T) {
	def := `CREATE INDEX idx ON t ()`
	cols := parseIndexColumns(def)
	if cols != nil {
		t.Errorf("Expected nil for empty parens, got %v", cols)
	}
}

func TestParseIndexColumns_ThreeCols(t *testing.T) {
	def := `CREATE INDEX idx ON t (a, b, c)`
	cols := parseIndexColumns(def)
	expected := []string{"a", "b", "c"}
	if len(cols) != len(expected) {
		t.Fatalf("Expected %d cols, got %d: %v", len(expected), len(cols), cols)
	}
	for i, col := range cols {
		if col != expected[i] {
			t.Errorf("Col[%d] = %q, want %q", i, col, expected[i])
		}
	}
}

func TestTrimSpace(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  hello  ", "hello"},
		{"\t\nfoo\t\n", "foo"},
		{"no trim", "no trim"},
		{"   ", ""},
		{"", ""},
		{"\t", ""},
		{"a", "a"},
	}
	for _, tt := range tests {
		got := trimSpace(tt.input)
		if got != tt.want {
			t.Errorf("trimSpace(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
