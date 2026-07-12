package tui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

// TokenType classifies SQL lexer tokens for syntax highlighting.
type TokenType int

const (
	TokenKeyword      TokenType = iota // SELECT, FROM, WHERE, JOIN, etc.
	TokenString                        // 'single quoted' or 'double quoted' strings
	TokenNumber                        // 42, 3.14, -1
	TokenComment                       // -- line comment or /* block */
	TokenOperator                      // =, <, >, +, -, *, /, etc.
	TokenPunctuation                   // , . ; ( )
	TokenIdentifier                    // table_name, column_name, unquoted identifiers
	TokenWhitespace                    // spaces, tabs
)

// token holds a single lexed SQL token with its type and value.
type token struct {
	typ   TokenType
	value string
}

// SQL keyword set — common SQL reserved words for highlighting.
var sqlKeywords = map[string]bool{
	"SELECT": true, "FROM": true, "WHERE": true, "INSERT": true,
	"INTO": true, "VALUES": true, "UPDATE": true, "SET": true,
	"DELETE": true, "CREATE": true, "TABLE": true, "ALTER": true,
	"DROP": true, "INDEX": true, "VIEW": true, "TRIGGER": true,
	"AND": true, "OR": true, "NOT": true, "IN": true,
	"IS": true, "NULL": true, "LIKE": true, "BETWEEN": true,
	"EXISTS": true, "AS": true, "ON": true, "JOIN": true,
	"LEFT": true, "RIGHT": true, "INNER": true, "OUTER": true,
	"FULL": true, "CROSS": true, "ORDER": true, "BY": true,
	"ASC": true, "DESC": true, "LIMIT": true, "OFFSET": true,
	"GROUP": true, "HAVING": true, "DISTINCT": true, "CASE": true,
	"WHEN": true, "THEN": true, "ELSE": true, "END": true,
	"UNION": true, "ALL": true, "EXCEPT": true, "INTERSECT": true,
	"WITH": true, "RECURSIVE": true, "RETURNING": true,
	"PRIMARY": true, "KEY": true, "FOREIGN": true, "REFERENCES": true,
	"CONSTRAINT": true, "UNIQUE": true, "CHECK": true, "DEFAULT": true,
	"CASCADE": true, "RESTRICT": true, "IF": true, "USING": true,
	"EXPLAIN": true, "ANALYZE": true, "VACUUM": true, "BEGIN": true,
	"COMMIT": true, "ROLLBACK": true, "SAVEPOINT": true,
	"GRANT": true, "REVOKE": true, "COUNT": true, "SUM": true,
	"AVG": true, "MIN": true, "MAX": true, "COALESCE": true,
	"CAST": true, "TRUE": true, "FALSE": true, "ANY": true,
	"SOME": true, "EXTRACT": true, "DATE": true, "TIME": true,
	"TIMESTAMP": true, "INTERVAL": true, "SCHEMA": true,
	"DATABASE": true, "TYPE": true, "ENUM": true, "SERIAL": true,
	"WINDOW": true, "PARTITION": true,
	"RANGE": true, "ROWS": true, "UNBOUNDED": true,
	"PRECEDING": true, "FOLLOWING": true, "CURRENT": true,
	"IDENTITY": true, "GENERATED": true,
	"STORED": true, "INCREMENT": true, "START": true,
	"OWNED": true, "TO": true, "ADD": true, "COLUMN": true,
	"RENAME": true, "DISABLE": true, "ENABLE": true,
	"REPLICA": true, "ALWAYS": true, "CLUSTER": true,
	"WITHOUT": true, "TRUNCATE": true, "REINDEX": true,
	"BIGINT": true, "INT": true, "INTEGER": true, "SMALLINT": true,
	"TEXT": true, "VARCHAR": true, "CHAR": true, "BOOLEAN": true,
	"FLOAT": true, "DOUBLE": true, "PRECISION": true, "REAL": true,
	"NUMERIC": true, "DECIMAL": true, "BLOB": true, "BYTEA": true,
	"JSON": true, "JSONB": true, "ARRAY": true,
	"CURSOR": true, "FETCH": true, "NEXT": true, "CLOSE": true,
	"DECLARE": true, "FUNCTION": true, "PROCEDURE": true,
	"RETURNS": true, "LANGUAGE": true, "PLPGSQL": true,
	"NOTHING": true, "DO": true, "CONFLICT": true, "EXCLUDED": true,
}

// isKeyword returns true if s is a SQL reserved word (case-insensitive).
func isKeyword(s string) bool {
	return sqlKeywords[strings.ToUpper(s)]
}

// highlightStyles maps token types to lipgloss styles.
func highlightStyles() map[TokenType]lipgloss.Style {
	return map[TokenType]lipgloss.Style{
		TokenKeyword: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#89B4FA")), // blue
		TokenString: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E3A1")), // green
		TokenNumber: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAB387")), // orange
		TokenComment: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")). // gray
			Italic(true),
		TokenOperator: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F5C2E7")), // pink
		TokenPunctuation: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#BAC2DE")), // light gray
		TokenIdentifier: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CDD6F4")), // white text
		TokenWhitespace: lipgloss.NewStyle(),
	}
}

// tokenizeSQL lexes a single line of SQL text into tokens.
// Returns a slice of tokens covering the entire input.
// This is a line-level tokenizer; multi-line comments and strings
// that span lines are handled at the rendering level.
func tokenizeSQL(line string) []token {
	var tokens []token
	runes := []rune(line)
	i := 0

	appendToken := func(t TokenType, val string) {
		if val == "" {
			return
		}
		tokens = append(tokens, token{typ: t, value: val})
	}

	for i < len(runes) {
		ch := runes[i]

		// Whitespace
		if ch == ' ' || ch == '\t' {
			start := i
			for i < len(runes) && (runes[i] == ' ' || runes[i] == '\t') {
				i++
			}
			appendToken(TokenWhitespace, string(runes[start:i]))
			continue
		}

		// Single-line comment: --
		if ch == '-' && i+1 < len(runes) && runes[i+1] == '-' {
			start := i
			for i < len(runes) {
				i++
			}
			appendToken(TokenComment, string(runes[start:i]))
			continue
		}

		// Block comment start: /*
		if ch == '/' && i+1 < len(runes) && runes[i+1] == '*' {
			start := i
			i += 2
			for i < len(runes) {
				if runes[i] == '*' && i+1 < len(runes) && runes[i+1] == '/' {
					i += 2
					break
				}
				i++
			}
			appendToken(TokenComment, string(runes[start:i]))
			continue
		}

		// String literals: 'single quoted'
		if ch == '\'' {
			start := i
			i++
			for i < len(runes) {
				if runes[i] == '\'' {
					i++
					if i >= len(runes) || runes[i] != '\'' {
						break // end of string (not an escaped quote)
					}
					i++ // skip escaped quote
				} else {
					i++
				}
			}
			appendToken(TokenString, string(runes[start:i]))
			continue
		}

		// String literals: "double quoted" identifiers
		if ch == '"' {
			start := i
			i++
			for i < len(runes) {
				if runes[i] == '"' {
					i++
					if i >= len(runes) || runes[i] != '"' {
						break
					}
					i++ // skip escaped quote
				} else {
					i++
				}
			}
			// Double-quoted strings are identifiers in SQL
			appendToken(TokenIdentifier, string(runes[start:i]))
			continue
		}

		// Numbers
		if unicode.IsDigit(ch) || (ch == '.' && i+1 < len(runes) && unicode.IsDigit(runes[i+1])) {
			start := i
			if ch == '.' {
				i++ // leading decimal point
			}
			hasDecimal := ch == '.'
			for i < len(runes) && (unicode.IsDigit(runes[i]) || runes[i] == '.') {
				if runes[i] == '.' {
					if hasDecimal {
						break
					}
					hasDecimal = true
				}
				i++
			}
			appendToken(TokenNumber, string(runes[start:i]))
			continue
		}

		// Operators and punctuation
		if isOperator(ch) || isPunctuation(ch) {
			if isPunctuation(ch) {
				appendToken(TokenPunctuation, string(ch))
				i++
			} else {
				// Multi-char operators: >=, <=, !=, <>, ||
				start := i
				i++
				if i < len(runes) {
					next := runes[i]
					if (ch == '!' && next == '=') ||
						(ch == '<' && (next == '=' || next == '>' || next == '<')) ||
						(ch == '>' && (next == '=' || next == '>')) ||
						(ch == '|' && next == '|') ||
						(ch == ':' && next == '=') ||
						(ch == '-' && next == '>') ||
						(ch == '=' && next == '=') {
						i++
					}
				}
				appendToken(TokenOperator, string(runes[start:i]))
			}
			continue
		}

		// Identifier or keyword (word characters)
		if isIdentChar(ch) {
			start := i
			for i < len(runes) && isIdentChar(runes[i]) {
				i++
			}
			word := string(runes[start:i])
			if isKeyword(word) {
				appendToken(TokenKeyword, word)
			} else {
				appendToken(TokenIdentifier, word)
			}
			continue
		}

		// Fallback: treat as identifier
		appendToken(TokenIdentifier, string(ch))
		i++
	}

	return tokens
}

// isOperator returns true for SQL operator characters.
func isOperator(ch rune) bool {
	switch ch {
	case '=', '<', '>', '!', '+', '-', '*', '/', '%', '|', '&', '^', '~', ':':
		return true
	}
	return false
}

// isPunctuation returns true for SQL punctuation characters.
func isPunctuation(ch rune) bool {
	switch ch {
	case ',', '.', ';', '(', ')', '[', ']':
		return true
	}
	return false
}

// isIdentChar returns true if the rune can be part of an unquoted SQL identifier.
func isIdentChar(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '$'
}

// highlightLine applies syntax highlighting to a single line of SQL text.
// Returns the line with ANSI escape codes for terminal color.
func highlightLine(line string) string {
	if line == "" {
		return ""
	}

	tokens := tokenizeSQL(line)
	styles := highlightStyles()

	var b strings.Builder
	for _, tok := range tokens {
		if style, ok := styles[tok.typ]; ok {
			b.WriteString(style.Render(tok.value))
		} else {
			b.WriteString(tok.value)
		}
	}
	return b.String()
}

// highlightSQL applies syntax highlighting and returns the highlighted text
// line by line. If cursorLine >= 0, that line gets a cursor indicator.
// showLineNumbers controls whether line numbers are prepended.
func highlightSQL(content string, cursorRow, cursorCol int, showLineNumbers bool, width int) string {
	if content == "" {
		return ""
	}

	lines := strings.Split(content, "\n")
	var b strings.Builder

	lineNumWidth := lenInt(len(lines))
	if lineNumWidth < 2 {
		lineNumWidth = 2
	}

	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))
	numStyleCursor := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F5C2E7")).
		Bold(true)
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#45475A"))

	for i, line := range lines {
		// Prompt character (like textarea's "❯ ")
		_, _ = b.WriteString(promptStyle.Render("> "))

		// Line number
		if showLineNumbers {
			numStr := formatLineNum(i+1, lineNumWidth)
			if i == cursorRow {
				_, _ = b.WriteString(numStyleCursor.Render(numStr))
			} else {
				_, _ = b.WriteString(numStyle.Render(numStr))
			}
		}

		// Highlighted content
		if i == cursorRow {
			// For cursor line, we need to handle cursor position
			// Split the line at cursor position
			cursorRunes := []rune(line)
			beforeCursor := string(cursorRunes[:min(cursorCol, len(cursorRunes))])
			cursorChar := ""
			afterCursor := ""
			if cursorCol < len(cursorRunes) {
				cursorChar = string(cursorRunes[cursorCol])
				afterCursor = string(cursorRunes[cursorCol+1:])
			} else {
				cursorChar = " "
			}

			// Highlight each segment separately
			beforeHighlighted := highlightLine(beforeCursor)
			cursorStyle := lipgloss.NewStyle().
				Background(lipgloss.Color("#89B4FA")).
				Foreground(lipgloss.Color("#1E1E2E"))
			afterHighlighted := highlightLine(afterCursor)

			_, _ = b.WriteString(beforeHighlighted)
			_, _ = b.WriteString(cursorStyle.Render(cursorChar))
			_, _ = b.WriteString(afterHighlighted)
		} else {
			_, _ = b.WriteString(highlightLine(line))
		}

		if i < len(lines)-1 {
			_ = b.WriteByte('\n')
		}
	}

	return b.String()
}

// formatLineNum pads a line number to the given width.
func formatLineNum(n, width int) string {
	s := strings.Repeat(" ", width-lenInt(n)) + itoa(n) + " "
	return s
}

// lenInt returns the number of digits in a non-negative integer.
func lenInt(n int) int {
	if n < 10 {
		return 1
	}
	if n < 100 {
		return 2
	}
	if n < 1000 {
		return 3
	}
	if n < 10000 {
		return 4
	}
	return 5
}

// itoa converts an int to a string (avoids importing strconv for a simple case).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		return "-" + string(digits)
	}
	return string(digits)
}
