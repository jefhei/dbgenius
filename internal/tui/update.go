package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jefhei/dbgenius/internal/ai"
)

// defaultQueryTimeout is the default timeout for SQL query execution.
const defaultQueryTimeout = 30 * time.Second

// Update handles all messages and returns the updated model.
func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.ready = true
		m.width = msg.Width
		m.height = msg.Height

		// Update child model sizes
		leftWidth := m.width / 4
		if leftWidth < 10 {
			leftWidth = 10
		}

		// Schema tree: left column, full height minus status bar
		m.schemaTree.SetSize(leftWidth, m.height-2)

		// Right side: editor (top 1/3) and data viewer (bottom 2/3)
		rightWidth := m.width - leftWidth - 3
		editorHeight := m.height / 3
		viewerHeight := m.height - editorHeight - 2

		if editorHeight < 3 {
			editorHeight = 3
		}
		if viewerHeight < 3 {
			viewerHeight = 3
		}

		m.sqlEditor.SetSize(rightWidth, editorHeight)
		m.dataViewer.SetSize(rightWidth, viewerHeight)
		return m, nil

	case TableSelectedMsg:
		// User selected a table in the schema tree
		if m.db != nil {
			cmd = m.dataViewer.SelectTable(m.db, msg.Schema, msg.Table)
			return m, cmd
		}
		return m, nil

	case ExecuteQueryMsg:
		// Query submitted from the editor
		if m.db == nil {
			return m, nil
		}
		if msg.Query == "" {
			return m, nil
		}

		// Cancel any previous running query
		if m.queryCancel != nil {
			m.queryCancel()
			m.queryCancel = nil
		}

		// Reset data viewer to loading state
		m.dataViewer.state = viewerLoading
		m.dataViewer.errMsg = ""

		// Create cancellable context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), defaultQueryTimeout)
		m.queryCancel = cancel
		m.isExecuting = true

		queryCopy := msg.Query
		dbCopy := m.db

		// Switch focus to results panel after executing
		m.focusedPanel = panelResults

		cmd = func() tea.Msg {
			defer cancel()

			result, err := dbCopy.ExecuteQuery(ctx, queryCopy)

			// Check for cancellation first
			if errors.Is(err, context.Canceled) {
				return queryCancelledMsg{}
			}

			// Check for deadline exceeded
			if errors.Is(err, context.DeadlineExceeded) {
				return tableDataErrorMsg{
					err:    fmt.Errorf("query timed out after %v", defaultQueryTimeout),
					schema: "",
					table:  "",
				}
			}

			if err != nil {
				return tableDataErrorMsg{err: err, schema: "", table: ""}
			}

			return tableDataLoadedMsg{
				columns: result.Columns,
				rows:    result.Rows,
				total:   int64(len(result.Rows)),
				schema:  "",
				table:   "Query Result",
			}
		}

		return m, cmd

	case SlashCommandMsg:
		// Handle slash commands from the editor
		switch msg.Command {
		case cmdHelp:
			// Show help text in results panel
			m.aiResponse = CommandHelpText()
			m.dataViewer.columns = []string{}
			m.dataViewer.rows = nil
			m.dataViewer.totalRows = 0
			m.focusedPanel = panelResults
			return m, nil

		case cmdExplain:
			// Take current editor content as the query to explain
			queryToExplain := msg.Args
			if queryToExplain == "" {
				queryToExplain = msg.EditorContent
			}
			if queryToExplain == "" {
				m.dataViewer.state = viewerError
				m.dataViewer.errMsg = "⚠️  No query to explain. Type a query or /explain <query>"
				m.focusedPanel = panelResults
				return m, nil
			}

			if m.aiClient == nil {
				m.dataViewer.state = viewerError
				m.dataViewer.errMsg = "⚠️  Ollama not configured. Set up in config file or env vars."
				m.focusedPanel = panelResults
				return m, nil
			}

			// Set loading state
			m.aiResponse = ""
			m.dataViewer.state = viewerLoading
			m.focusedPanel = panelResults

			// Build schema context
			var schemaCtx string
			if m.db != nil {
				ctx := context.Background()
				schemaInfo, err := m.db.Introspect(ctx)
				if err == nil && schemaInfo != nil && m.schemaContextBuilder != nil {
					schemaCtx = m.schemaContextBuilder.BuildContext(schemaInfo, m.db.GetIntrospector())
				}
			}

			// Build prompt and send to Ollama in a goroutine
			aiClient := m.aiClient
			prompt := ai.BuildExplainPrompt(schemaCtx, queryToExplain)
			cmd = func() tea.Msg {
				response, err := aiClient.Generate(context.Background(), "", prompt)
				return aiResponseMsg{
					response: response,
					err:      err,
					command:  cmdExplain,
				}
			}
			return m, cmd

		case cmdSuggest:
			if m.aiClient == nil {
				m.dataViewer.state = viewerError
				m.dataViewer.errMsg = "⚠️  Ollama not configured. Set up in config file or env vars."
				m.focusedPanel = panelResults
				return m, nil
			}
			if msg.Args == "" {
				m.dataViewer.state = viewerError
				m.dataViewer.errMsg = "⚠️  Usage: /suggest <description of what you need>"
				m.focusedPanel = panelResults
				return m, nil
			}

			// Set loading state in results while generating
			m.aiResponse = ""
			m.dataViewer.state = viewerLoading
			m.focusedPanel = panelResults

			// Build schema context
			var schemaCtx string
			if m.db != nil {
				ctx := context.Background()
				schemaInfo, err := m.db.Introspect(ctx)
				if err == nil && schemaInfo != nil && m.schemaContextBuilder != nil {
					schemaCtx = m.schemaContextBuilder.BuildContext(schemaInfo, m.db.GetIntrospector())
				}
			}

			// Build prompt and send to Ollama
			aiClient := m.aiClient
			userRequest := msg.Args
			prompt := ai.BuildSuggestPrompt(schemaCtx, userRequest)
			cmd = func() tea.Msg {
				response, err := aiClient.Generate(context.Background(), "", prompt)
				// Extract SQL from the response
				sql := extractSQL(response)
				if sql == "" {
					sql = "/* AI did not return a valid SQL query */\n" + response
				}
				return aiSuggestionMsg{
					query: sql,
					err:   err,
				}
			}
			return m, cmd

		case cmdOptimize:
			if m.aiClient == nil {
				m.dataViewer.state = viewerError
				m.dataViewer.errMsg = "⚠️  Ollama not configured. Set up in config file or env vars."
				m.focusedPanel = panelResults
				return m, nil
			}
			m.focusedPanel = panelResults
			return m, nil

		case cmdInvalid:
			m.dataViewer.state = viewerError
			m.dataViewer.errMsg = InvalidCommandError(msg.RawInput)
			m.focusedPanel = panelResults
			return m, nil
		}
		return m, nil

	case aiResponseMsg:
		// Handle AI response (display in results panel)
		if msg.err != nil {
			m.aiResponse = ""
			m.dataViewer.state = viewerError
			m.dataViewer.errMsg = ai.FriendlyError(msg.err)
		} else {
			m.aiResponse = msg.response
			m.dataViewer.columns = []string{}
			m.dataViewer.rows = nil
			m.dataViewer.totalRows = 0
		}
		m.focusedPanel = panelResults
		return m, nil

	case aiSuggestionMsg:
		// Handle AI suggestion (place in editor)
		if msg.err != nil {
			m.aiResponse = ""
			m.dataViewer.state = viewerError
			m.dataViewer.errMsg = ai.FriendlyError(msg.err)
			m.focusedPanel = panelResults
		} else {
			m.sqlEditor.SetValue(msg.query)
			m.focusedPanel = panelEditor
		}
		return m, nil

	case queryCancelledMsg:
		// Query was cancelled — reset state
		m.isExecuting = false
		m.queryCancel = nil
		return m, nil

	case tableDataLoadedMsg:
		// Data loaded — reset executing state and delegate to data viewer
		m.isExecuting = false
		m.queryCancel = nil
		var dvCmd tea.Cmd
		m.dataViewer, dvCmd = m.dataViewer.Update(msg)
		cmds = append(cmds, dvCmd)

	case tableDataErrorMsg:
		// Error loading data — reset executing state and delegate to data viewer
		m.isExecuting = false
		m.queryCancel = nil
		var dvCmd tea.Cmd
		m.dataViewer, dvCmd = m.dataViewer.Update(msg)
		cmds = append(cmds, dvCmd)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			// If a query is executing, cancel it instead of quitting
			if m.isExecuting && m.queryCancel != nil {
				m.queryCancel()
				m.queryCancel = nil
				return m, nil
			}
			// If help is open, close it first
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
			return m, tea.Quit

		case "q":
			// Only quit if not executing a query
			if m.isExecuting {
				return m, nil
			}
			// If help is open, close it first
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
			return m, tea.Quit

		case "?":
			m.showHelp = !m.showHelp
			return m, nil

		case "tab", "ctrl+w":
			// Cycle forward through panels
			m.focusedPanel = m.nextPanel()
			m.syncFocus(&cmds)
			return m, nil

		case "shift+tab", "ctrl+shift+tab":
			// Cycle backward through panels
			prev := (int(m.focusedPanel) - 1 + int(panelCount)) % int(panelCount)
			m.focusedPanel = panel(prev)
			m.syncFocus(&cmds)
			return m, nil
		}

		// If help is open, only handle close
		if m.showHelp {
			return m, nil
		}
	}

	if m.showHelp {
		return m, nil
	}

	// Delegate to focused panel
	switch m.focusedPanel {
	case panelSchemaTree:
		var treeCmd tea.Cmd
		m.schemaTree, treeCmd = m.schemaTree.Update(msg)
		cmds = append(cmds, treeCmd)

	case panelEditor:
		// Handle up/down in command mode for history navigation
		if keyMsg, ok := msg.(tea.KeyMsg); ok && m.sqlEditor.mode == editorCommand {
			switch keyMsg.String() {
			case "up":
				m.sqlEditor.HistoryNavUp()
				return m, nil
			case "down":
				m.sqlEditor.HistoryNavDown()
				return m, nil
			}
		}

		var editorCmd tea.Cmd
		m.sqlEditor, editorCmd = m.sqlEditor.Update(msg)
		cmds = append(cmds, editorCmd)

	case panelResults:
		var dvCmd tea.Cmd
		m.dataViewer, dvCmd = m.dataViewer.Update(msg)
		cmds = append(cmds, dvCmd)
	}

	// Always let the data viewer handle non-key messages (tableDataLoadedMsg, etc.)
	// even when not focused, so loading completes correctly
	if !isKeyMsg(msg) && m.focusedPanel != panelResults {
		var dvCmd tea.Cmd
		m.dataViewer, dvCmd = m.dataViewer.Update(msg)
		cmds = append(cmds, dvCmd)
	}

	return m, tea.Batch(cmds...)
}

// nextPanel returns the next panel in the focus cycle.
func (m RootModel) nextPanel() panel {
	return panel((int(m.focusedPanel) + 1) % int(panelCount))
}

// syncFocus updates focus state of child models when focus panel changes.
func (m *RootModel) syncFocus(cmds *[]tea.Cmd) {
	// Focus the newly selected panel, blur others
	switch m.focusedPanel {
	case panelSchemaTree:
		m.sqlEditor.Blur()
	case panelEditor:
		*cmds = append(*cmds, m.sqlEditor.Focus())
	case panelResults:
		m.sqlEditor.Blur()
	}
}

// isKeyMsg checks if a message is a key press (to avoid data viewer handling
// keyboard input when it's not focused).
func isKeyMsg(msg tea.Msg) bool {
	_, ok := msg.(tea.KeyMsg)
	return ok
}

// executeQueryBare runs a SQL query and returns the result as a Bubble Tea message.
func executeQueryBare(query string) tea.Msg {
	return ExecuteQueryMsg{Query: query}
}

// renderError creates a formatted error string.
func renderError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("✗ %s", err.Error())
}

// extractSQL attempts to extract a SQL query from an AI response.
// It looks for SQL code blocks (```sql ... ```) or standalone SQL statements.
// Returns the full response if no SQL pattern is found.
func extractSQL(response string) string {
	if response == "" {
		return ""
	}

	// Try to extract from a SQL code block first
	lower := strings.ToLower(response)

	// Pattern: ```sql ... ```
	const sqlBlockStart = "```sql"
	const sqlBlockEnd = "```"

	startIdx := strings.Index(lower, sqlBlockStart)
	if startIdx >= 0 {
		// Move past the opening ```sql
		contentStart := startIdx + len(sqlBlockStart)
		endIdx := strings.Index(response[contentStart:], sqlBlockEnd)
		if endIdx >= 0 {
			sql := strings.TrimSpace(response[contentStart : contentStart+endIdx])
			return sql
		}
		// Fallback: take everything after the opening tag
		sql := strings.TrimSpace(response[contentStart:])
		if sql != "" {
			return sql
		}
	}

	// Pattern: ``` ... ``` (any code block) - try to extract it
	startIdx = strings.Index(lower, "```")
	if startIdx >= 0 {
		contentStart := startIdx + 3
		// Skip past any language identifier on the same line
		rest := response[contentStart:]
		newlineIdx := strings.IndexByte(rest, '\n')
		if newlineIdx >= 0 {
			contentStart += newlineIdx + 1
		}
		endIdx := strings.Index(response[contentStart:], sqlBlockEnd)
		if endIdx >= 0 {
			sql := strings.TrimSpace(response[contentStart : contentStart+endIdx])
			if sql != "" {
				return sql
			}
		}
	}

	// No code block found — return the full response as-is
	return strings.TrimSpace(response)
}
