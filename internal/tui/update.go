package tui

import (
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
		m.schemaTree.SetSize(leftWidth, m.height-2)

		// Data viewer gets remaining space
		rightWidth := m.width - leftWidth - 3
		rightBottomHeight := m.height - (m.height / 3) - 2
		if rightBottomHeight < 3 {
			rightBottomHeight = 3
		}
		m.dataViewer.SetSize(rightWidth, rightBottomHeight)
		return m, nil

	case TableSelectedMsg:
		// User selected a table in the schema tree
		if m.db != nil {
			cmd = m.dataViewer.SelectTable(m.db, msg.Schema, msg.Table)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			next := (int(m.focusedPanel) + 1) % int(panelCount)
			m.focusedPanel = panel(next)
			return m, nil

		case "shift+tab":
			prev := (int(m.focusedPanel) - 1 + int(panelCount)) % int(panelCount)
			m.focusedPanel = panel(prev)
			return m, nil
		}
	}

	// Delegate to focused panel
	switch m.focusedPanel {
	case panelSchemaTree:
		var treeCmd tea.Cmd
		m.schemaTree, treeCmd = m.schemaTree.Update(msg)
		cmds = append(cmds, treeCmd)
	case panelResults:
		var dvCmd tea.Cmd
		m.dataViewer, dvCmd = m.dataViewer.Update(msg)
		cmds = append(cmds, dvCmd)
	}

	// Always let the data viewer handle its own messages (tableDataLoadedMsg, etc.)
	// even when not focused, so loading completes correctly
	if !isKeyMsg(msg) {
		var dvCmd tea.Cmd
		m.dataViewer, dvCmd = m.dataViewer.Update(msg)
		cmds = append(cmds, dvCmd)
	}

	return m, tea.Batch(cmds...)
}

// isKeyMsg checks if a message is a key press (to avoid data viewer handling
// keyboard input when it's not focused).
func isKeyMsg(msg tea.Msg) bool {
	_, ok := msg.(tea.KeyMsg)
	return ok
}
