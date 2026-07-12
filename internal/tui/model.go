package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jefhei/dbgenius/internal/ai"
	"github.com/jefhei/dbgenius/internal/db"
)

// queryCancelledMsg is sent when a running query is cancelled by the user.
type queryCancelledMsg struct{}

// aiResponseMsg carries the AI's response to a slash command.
type aiResponseMsg struct {
	response string
	err      error
	command  SlashCommand
}

// aiSuggestionMsg carries an AI-suggested query to be placed in the editor.
type aiSuggestionMsg struct {
	query string
	err   error
}

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

	// Async query execution state
	isExecuting bool
	queryCancel context.CancelFunc

	// AI client for slash commands
	aiClient         *ai.Client
	schemaContextBuilder *ai.SchemaContextBuilder

	// AI response state
	aiResponse string
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

// SetAIClient sets the AI client for slash commands.
func (m *RootModel) SetAIClient(client *ai.Client) {
	m.aiClient = client
	m.schemaContextBuilder = ai.NewSchemaContextBuilder()
}
