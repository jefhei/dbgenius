package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles all messages and returns the updated model.
func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.ready = true
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			// Cycle focus through panels
			next := (int(m.focusedPanel) + 1) % int(panelCount)
			m.focusedPanel = panel(next)
			return m, nil

		case "shift+tab":
			prev := (int(m.focusedPanel) - 1 + int(panelCount)) % int(panelCount)
			m.focusedPanel = panel(prev)
			return m, nil
		}
	}

	return m, nil
}
