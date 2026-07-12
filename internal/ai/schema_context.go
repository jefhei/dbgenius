package ai

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jefhei/dbgenius/internal/db"
)

// SchemaContextBuilder builds compact text representations of database schemas
// suitable for inclusion in LLM prompts.
type SchemaContextBuilder struct {
	mu             sync.RWMutex
	cachedContext  string
	cachedSchema   string // cache key: schema hash/name
	cachedAt       time.Time
	cacheTTL       time.Duration
	maxContextLen  int // max characters for the built context
}

// NewSchemaContextBuilder creates a new schema context builder.
func NewSchemaContextBuilder() *SchemaContextBuilder {
	return &SchemaContextBuilder{
		cacheTTL:      5 * time.Minute,
		maxContextLen: 8000, // ~2K tokens for reasonable LLM context window
	}
}

// BuildContext creates a compact text representation of the given schema.
// The text format is designed to be easily consumed by LLMs — structured,
// concise, and without unnecessary formatting characters.
func (b *SchemaContextBuilder) BuildContext(schemaInfo *db.SchemaInfo, introspector db.SchemaIntrospector) string {
	if schemaInfo == nil {
		return ""
	}

	// Check cache
	cacheKey := b.cacheKey(schemaInfo)
	b.mu.RLock()
	if b.cachedContext != "" && b.cachedSchema == cacheKey && time.Since(b.cachedAt) < b.cacheTTL {
		ctx := b.cachedContext
		b.mu.RUnlock()
		return ctx
	}
	b.mu.RUnlock()

	// Build fresh context
	ctx := b.build(schemaInfo, introspector)

	// Cache it
	b.mu.Lock()
	b.cachedContext = ctx
	b.cachedSchema = cacheKey
	b.cachedAt = time.Now()
	b.mu.Unlock()

	return ctx
}

// InvalidateCache forces a rebuild on the next call.
func (b *SchemaContextBuilder) InvalidateCache() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.cachedContext = ""
	b.cachedSchema = ""
}

// build creates the compact text representation from schema metadata.
func (b *SchemaContextBuilder) build(schemaInfo *db.SchemaInfo, _ db.SchemaIntrospector) string {
	var buf strings.Builder

	// Database overview
	buf.WriteString("=== DATABASE SCHEMA ===\n\n")

	// List schemas
	schemas := schemaInfo.Schemas
	if len(schemas) == 0 {
		// Collect from tables map
		for schema := range schemaInfo.Tables {
			schemas = append(schemas, schema)
		}
	}

	for _, schema := range schemas {
		tables := schemaInfo.Tables[schema]
		if len(tables) == 0 {
			continue
		}

		buf.WriteString(fmt.Sprintf("--- Schema: %s (%d tables) ---\n", schema, len(tables)))

		for _, extTable := range tables {
			t := extTable.Table
			buf.WriteString(fmt.Sprintf("\nTABLE %s.%s\n", t.Schema, t.Name))

			// Columns
			if len(t.Columns) > 0 {
				buf.WriteString("  Columns:\n")
				for _, col := range t.Columns {
					nullable := ""
					if col.Nullable {
						nullable = " NULL"
					}
					pk := ""
					if col.IsPrimaryKey {
						pk = " PK"
					}
					def := ""
					if col.DefaultValue != nil && *col.DefaultValue != "" {
						def = fmt.Sprintf(" DEFAULT %s", *col.DefaultValue)
					}
					buf.WriteString(fmt.Sprintf("    - %s: %s%s%s%s\n",
						col.Name, col.DataType, nullable, pk, def))
				}
			}

			// Foreign Keys
			if len(extTable.ForeignKeys) > 0 {
				buf.WriteString("  Foreign Keys:\n")
				for _, fk := range extTable.ForeignKeys {
					buf.WriteString(fmt.Sprintf("    - %s -> %s.%s.%s",
						fk.ColumnName, fk.RefSchema, fk.RefTable, fk.RefColumnName))
					if fk.ConstraintName != "" {
						buf.WriteString(fmt.Sprintf(" (%s)", fk.ConstraintName))
					}
					buf.WriteString("\n")
				}
			}

			// Indexes
			if len(extTable.Indexes) > 0 {
				buf.WriteString("  Indexes:\n")
				for _, idx := range extTable.Indexes {
					unique := ""
					if idx.IsUnique {
						unique = " UNIQUE"
					}
					buf.WriteString(fmt.Sprintf("    - %s%s ON (%s)\n",
						idx.Name, unique, strings.Join(idx.Columns, ", ")))
				}
			}
		}
		buf.WriteString("\n")
	}

	// Get the built string and trim if needed
	result := buf.String()
	if len(result) > b.maxContextLen {
		result = b.truncatePreserveStructure(result)
	}

	return result
}

// truncatePreserveStructure truncates the schema context from the end,
// preserving key structural information.
func (b *SchemaContextBuilder) truncatePreserveStructure(s string) string {
	if len(s) <= b.maxContextLen {
		return s
	}

	// Keep the header and first part, truncate the rest
	// Aim for about 80% of max, then add a truncation notice
	targetLen := b.maxContextLen - 200 // leave room for truncation notice
	if targetLen < 0 {
		targetLen = b.maxContextLen / 2
	}

	// Try to cut at a table boundary
	cutPos := strings.LastIndex(s[:targetLen], "\nTABLE ")
	if cutPos > 0 {
		result := s[:cutPos] + "\n\n[... schema truncated due to size ...]\n"
		return result
	}

	// Fallback: cut at a section boundary
	cutPos = strings.LastIndex(s[:targetLen], "\n--- Schema:")
	if cutPos > 0 {
		result := s[:cutPos] + "\n\n[... schema truncated due to size ...]\n"
		return result
	}

	// Last resort: hard truncate
	return s[:b.maxContextLen-100] + "\n\n[... schema truncated ...]\n"
}

// cacheKey generates a simple cache key from the schema info.
func (b *SchemaContextBuilder) cacheKey(schemaInfo *db.SchemaInfo) string {
	var keys []string
	for schema := range schemaInfo.Tables {
		keys = append(keys, schema)
	}
	// Use the sorted schema names joined together
	return strings.Join(keys, ",")
}
