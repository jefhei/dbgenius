package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles all messages and returns the updated model.
func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

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
		m.schemaTree, cmd = m.schemaTree.Update(msg)
	}

	return m, cmd
}
