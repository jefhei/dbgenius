package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// RootModel is the top-level model managing all child models.
type RootModel struct {
	ready bool
	width int
	height int

	focusedPanel panel
}

// panel identifies which panel currently has focus.
type panel int

const (
	panelSchemaTree panel = iota
	panelEditor
	panelResults
	panelCount
)

// NewRootModel creates a new root model with default state.
func NewRootModel() RootModel {
	return RootModel{
		ready:        false,
		width:        80,
		height:       24,
		focusedPanel: panelSchemaTree,
	}
}

// Init initializes the model and returns any initial commands.
func (m RootModel) Init() tea.Cmd {
	return nil
}
