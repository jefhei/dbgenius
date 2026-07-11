package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

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
		// Execute the query against the current database connection
		if m.db == nil {
			return m, nil
		}
		if msg.Query == "" {
			return m, nil
		}

		// Delegate to data viewer for execution
		// (uses existing load mechanism — M2.5 will provide proper async handling)
		queryCopy := msg.Query
		dbCopy := m.db
		cmd = func() tea.Msg {
			result, err := dbCopy.ExecuteQuery(context.Background(), queryCopy)
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

		// Switch focus to results panel after executing
		m.focusedPanel = panelResults
		return m, cmd

	case tableDataLoadedMsg:
		// Data loaded — delegate to data viewer
		var dvCmd tea.Cmd
		m.dataViewer, dvCmd = m.dataViewer.Update(msg)
		cmds = append(cmds, dvCmd)

	case tableDataErrorMsg:
		// Error loading data — delegate to data viewer
		var dvCmd tea.Cmd
		m.dataViewer, dvCmd = m.dataViewer.Update(msg)
		cmds = append(cmds, dvCmd)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			// If help is open, close it first
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
			return m, tea.Quit

		case "?":
			m.showHelp = !m.showHelp
			return m, nil

		case "tab":
			// Cycle forward through panels
			m.focusedPanel = m.nextPanel()
			m.syncFocus()
			return m, nil

		case "shift+tab":
			// Cycle backward through panels
			prev := (int(m.focusedPanel) - 1 + int(panelCount)) % int(panelCount)
			m.focusedPanel = panel(prev)
			m.syncFocus()
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
func (m *RootModel) syncFocus() {
	// Focus the newly selected panel, blur others
	switch m.focusedPanel {
	case panelSchemaTree:
		m.sqlEditor.Blur()
	case panelEditor:
		cmd := m.sqlEditor.Focus()
		if cmd != nil {
			go func() { cmd() }()
		}
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
