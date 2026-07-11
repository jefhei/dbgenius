package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jefhei/dbgenius/internal/db"
)

// RootModel is the top-level model managing all child models.
type RootModel struct {
	ready bool
	width int
	height int

	focusedPanel panel
	schemaTree   TreeModel
	sqlEditor    SQLEditorModel
	dataViewer   DataViewerModel

	// Connected database for data browsing
	db *db.IntrospectedBackend

	// Show help overlay
	showHelp bool
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
		schemaTree:   NewTreeModel(),
		sqlEditor:    NewSQLEditorModel(),
		dataViewer:   NewDataViewerModel(),
		showHelp:     false,
	}
}

// Init initializes the model and returns any initial commands.
func (m RootModel) Init() tea.Cmd {
	return nil
}

// SetDB sets the connected database and propagates it to child components.
func (m *RootModel) SetDB(database *db.IntrospectedBackend) {
	m.db = database
	m.schemaTree.SetDB(database, database.GetType())
}
