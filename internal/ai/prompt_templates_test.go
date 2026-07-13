package ai

import (
	"testing"
)

func TestDefaultTemplates(t *testing.T) {
	templates := DefaultTemplates()
	if templates.Explain == "" {
		t.Error("Explain template should not be empty")
	}
	if templates.Suggest == "" {
		t.Error("Suggest template should not be empty")
	}
	if templates.Optimize == "" {
		t.Error("Optimize template should not be empty")
	}
}

func TestBuildExplainPrompt(t *testing.T) {
	schemaCtx := "TABLE users (id INT, name TEXT)"
	query := "SELECT * FROM users"

	prompt := BuildExplainPrompt(schemaCtx, query)
	if prompt == "" {
		t.Fatal("Expected non-empty prompt")
	}
	if !contains(prompt, schemaCtx) {
		t.Error("Prompt should contain schema context")
	}
	if !contains(prompt, query) {
		t.Error("Prompt should contain query")
	}
	if !contains(prompt, "QUERY TO EXPLAIN") {
		t.Error("Prompt should have section header")
	}
	if !contains(prompt, "=== SCHEMA CONTEXT ===") {
		t.Error("Prompt should have schema context header")
	}
}

func TestBuildSuggestPrompt(t *testing.T) {
	schemaCtx := "TABLE orders (id INT, product TEXT)"
	request := "show all orders"

	prompt := BuildSuggestPrompt(schemaCtx, request)
	if prompt == "" {
		t.Fatal("Expected non-empty prompt")
	}
	if !contains(prompt, schemaCtx) {
		t.Error("Prompt should contain schema context")
	}
	if !contains(prompt, request) {
		t.Error("Prompt should contain user request")
	}
	if !contains(prompt, "USER REQUEST") {
		t.Error("Prompt should have USER REQUEST header")
	}
}

func TestBuildOptimizePrompt(t *testing.T) {
	schemaCtx := "TABLE products (id INT, price DECIMAL)"
	query := "SELECT * FROM products WHERE price > 100"

	prompt := BuildOptimizePrompt(schemaCtx, query)
	if prompt == "" {
		t.Fatal("Expected non-empty prompt")
	}
	if !contains(prompt, schemaCtx) {
		t.Error("Prompt should contain schema context")
	}
	if !contains(prompt, query) {
		t.Error("Prompt should contain query")
	}
	if !contains(prompt, "QUERY TO OPTIMIZE") {
		t.Error("Prompt should have QUERY TO OPTIMIZE header")
	}
}

func TestBuildExplainPrompt_EmptyQuery(t *testing.T) {
	prompt := BuildExplainPrompt("schema context", "")
	if prompt == "" {
		t.Error("Expected non-empty prompt even with empty query")
	}
}
