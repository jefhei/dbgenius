package tui

import (
	"context"
	"time"

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

// aiStreamTokenMsg is sent for each token during streaming AI generation.
type aiStreamTokenMsg struct {
	token        string
	done         bool
	fullResponse string
	err          error
	command      SlashCommand
	buffer       *streamBuffer // for polling subsequent tokens
}

// streamBuffer holds streaming tokens that arrive asynchronously from an SSE stream.
// Each token is delivered to the Bubble Tea event loop via channel polling.
type streamBuffer struct {
	ch      chan aiStreamTokenMsg
	stopped bool
}

// ollamaHealthCheckMsg triggers a periodic check of Ollama availability.
type ollamaHealthCheckMsg struct{}


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

	// Ollama connection health
	ollamaAvailable     bool
	ollamaCheckInterval time.Duration

	// Whether to stop the background health check ticker
	ollamaChecking bool
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
	// If we have an AI client, start periodic health checks
	if m.aiClient != nil {
		m.ollamaChecking = true
		return m.startOllamaHealthCheck()
	}
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
	m.ollamaCheckInterval = 30 * time.Second
}

// startOllamaHealthCheck returns a command that checks Ollama availability
// and schedules subsequent checks periodically.
func (m RootModel) startOllamaHealthCheck() tea.Cmd {
	if m.aiClient == nil {
		m.ollamaAvailable = false
		return nil
	}

	return tea.Tick(m.ollamaCheckInterval, func(t time.Time) tea.Msg {
		return ollamaHealthCheckMsg{}
	})
}

// performOllamaCheck runs the actual health check against the Ollama server.
func (m RootModel) performOllamaCheck() tea.Msg {
	if m.aiClient == nil {
		return nil
	}

	err := m.aiClient.HealthCheck(context.Background())
	m.ollamaAvailable = (err == nil)
	return nil
}
