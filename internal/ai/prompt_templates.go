package ai

import "fmt"

// PromptTemplates holds the prompt strings for AI-powered commands.
// Each template uses fmt-style placeholders:
//   %s — schema context (from SchemaContextBuilder)
//   %s — query or user request
type PromptTemplates struct {
	Explain  string
	Suggest  string
	Optimize string
}

// DefaultTemplates returns the default set of prompt templates.
func DefaultTemplates() PromptTemplates {
	return PromptTemplates{
		Explain: explainTemplate,
		Suggest: suggestTemplate,
		Optimize: optimizeTemplate,
	}
}

// BuildExplainPrompt creates the prompt for the /explain command.
func BuildExplainPrompt(schemaContext, query string) string {
	return fmt.Sprintf(explainTemplate, schemaContext, query)
}

// BuildSuggestPrompt creates the prompt for the /suggest command.
func BuildSuggestPrompt(schemaContext, userRequest string) string {
	return fmt.Sprintf(suggestTemplate, schemaContext, userRequest)
}

// BuildOptimizePrompt creates the prompt for the /optimize command.
func BuildOptimizePrompt(schemaContext, query string) string {
	return fmt.Sprintf(optimizeTemplate, schemaContext, query)
}

const explainTemplate = `You are a SQL expert assistant. Your task is to explain the given SQL query in clear, simple terms.

=== SCHEMA CONTEXT ===
%s

=== QUERY TO EXPLAIN ===
%s

Please provide a thorough explanation covering:
1. What the query does overall
2. Which tables and columns are involved
3. How joins, filters, aggregations work
4. The expected output structure
5. Any notable patterns or techniques used

Format your response with:
- **bold** for key concepts
- Code blocks for SQL fragments
- Bullet points for lists`

const suggestTemplate = `You are a SQL expert assistant. Your task is to suggest a SQL query based on a user's natural language request.

=== SCHEMA CONTEXT ===
%s

=== USER REQUEST ===
%s

Please:
1. Write a SQL query that fulfills the request
2. Explain briefly what the query does
3. Note any assumptions you made about the schema

Important:
- Use only tables and columns that exist in the schema context above
- Use proper SQL syntax for the database type implied by the schema
- Handle NULLs appropriately
- Add meaningful comments to the query
- Output the SQL in a code block`

const optimizeTemplate = `You are a SQL performance expert. Your task is to analyze and optimize the given SQL query.

=== SCHEMA CONTEXT ===
%s

=== QUERY TO OPTIMIZE ===
%s

Please provide:
1. **Performance Analysis**: Identify current performance issues
2. **Optimized Query**: Provide an improved version of the query
3. **What Changed**: List the specific changes made
4. **Why It's Faster**: Explain how each change improves performance

Focus on:
- Missing indexes that would help
- Inefficient JOIN order
- Suboptimal WHERE clause usage
- Unnecessary subqueries or CTEs
- Better use of aggregations
- Avoiding full table scans
- Using appropriate LIMIT/TOP clauses

Format the optimized query in a code block and explain changes in plain text.`
