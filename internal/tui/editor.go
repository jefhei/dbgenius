package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// EditorMode represents the current editing mode.
type EditorMode int

const (
	editorInsert EditorMode = iota
	editorCommand
)

// ExecuteQueryMsg is sent when the user presses Ctrl+Enter to run a query.
type ExecuteQueryMsg struct {
	Query string
}

// SQLEditorModel implements a multi-line SQL editor with vim-like mode toggling,
// Ctrl+Enter execution, Tab indentation, query history, and SQL syntax highlighting.
type SQLEditorModel struct {
	textarea     textarea.Model
	mode         EditorMode
	width        int
	height       int
	focused      bool

	// Query history
	history      []string
	historyPos   int // -1 means not browsing history
	maxHistory   int

	// Viewport for scrolling highlighted content
	viewport viewport.Model

	// Scroll tracking (persists across View() value copies)
	scrollOffset int

	// Styles
	focusedStyle   lipgloss.Style
	blurredStyle   lipgloss.Style
	modeIndicator  lipgloss.Style
	placeholder    string

	// Cached view
	cachedView string
	dirty      bool
}

// NewSQLEditorModel creates a new SQL editor with default settings.
func NewSQLEditorModel() SQLEditorModel {
	ta := textarea.New()
	ta.Placeholder = "Enter SQL here..."
	ta.Prompt = "❯ "
	ta.ShowLineNumbers = true
	ta.CharLimit = 0 // no limit
	ta.MaxHeight = 1000
	ta.MaxWidth = 200
	ta.SetWidth(60)
	ta.SetHeight(10)

	// Use default styles
	focusedStyle, blurredStyle := textarea.DefaultStyles()
	ta.FocusedStyle = focusedStyle
	ta.BlurredStyle = blurredStyle

	// Create viewport for scrolling highlighted content
	vp := viewport.New(60, 10)
	vp.Style = lipgloss.NewStyle()

	return SQLEditorModel{
		textarea:   ta,
		mode:       editorInsert,
		focused:    false,
		history:    make([]string, 0, 100),
		historyPos: -1,
		maxHistory: 100,
		viewport:   vp,
		placeholder: "  Enter SQL here...\n  Ctrl+Enter to execute  |  Esc: command mode  |  i: insert mode",
		focusedStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E3A1")),
		blurredStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")),
		modeIndicator: lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1),
		dirty: true,
	}
}

// Init implements tea.Model.
func (m SQLEditorModel) Init() tea.Cmd {
	return m.textarea.Focus()
}

// Value returns the current editor content.
func (m SQLEditorModel) Value() string {
	return m.textarea.Value()
}

// SetValue sets the editor content (e.g., for /suggest replacement).
func (m *SQLEditorModel) SetValue(s string) {
	m.textarea.SetValue(s)
	m.textarea.SetCursor(len([]rune(s))) // move cursor to end
	m.dirty = true
}

// Focused returns whether the editor has focus.
func (m SQLEditorModel) Focused() bool {
	return m.focused
}

// SetSize updates the editor dimensions.
func (m *SQLEditorModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	// Leave room for mode indicator line and border
	taHeight := height - 2
	if taHeight < 3 {
		taHeight = 3
	}
	m.textarea.SetWidth(width - 2)
	m.textarea.SetHeight(taHeight)
	m.viewport.Width = width - 2
	m.viewport.Height = taHeight
	m.dirty = true
}

// Focus gives focus to the editor.
func (m *SQLEditorModel) Focus() tea.Cmd {
	m.focused = true
	return m.textarea.Focus()
}

// Blur removes focus from the editor.
func (m *SQLEditorModel) Blur() {
	m.focused = false
	m.textarea.Blur()
}

// GetMode returns the current editor mode.
func (m SQLEditorModel) GetMode() EditorMode {
	return m.mode
}

// Reset clears the editor content.
func (m *SQLEditorModel) Reset() {
	m.textarea.Reset()
	m.dirty = true
}

// AddToHistory saves a query to history.
func (m *SQLEditorModel) AddToHistory(query string) {
	query = strings.TrimSpace(query)
	if query == "" {
		return
	}
	// Don't add duplicate of last entry
	if len(m.history) > 0 && m.history[len(m.history)-1] == query {
		return
	}
	m.history = append(m.history, query)
	// Trim if over max
	if len(m.history) > m.maxHistory {
		m.history = m.history[len(m.history)-m.maxHistory:]
	}
	m.historyPos = -1
}

// Update handles messages and keyboard input for the editor.
func (m SQLEditorModel) Update(msg tea.Msg) (SQLEditorModel, tea.Cmd) {
	m.dirty = true

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		// Handle global shortcuts regardless of mode
		switch msg.String() {
		case "ctrl+c":
			// Pass through — let root model handle quit
			return m, nil
		}

		if !m.focused {
			// Don't process keystrokes when not focused
			return m, nil
		}

		switch m.mode {
		case editorInsert:
			return m.handleInsertModeKey(msg)
		case editorCommand:
			return m.handleCommandModeKey(msg)
		}
	}

	// Default: delegate to textarea
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	// Update scroll position based on new cursor location
	m.updateScrollPosition(m.textarea.Line())
	return m, cmd
}

// handleInsertModeKey processes keys in insert mode.
func (m SQLEditorModel) handleInsertModeKey(msg tea.KeyMsg) (SQLEditorModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Switch to command mode
		m.mode = editorCommand
		return m, nil

	case "ctrl+enter":
		// Execute query or slash command
		content := m.textarea.Value()
		query := strings.TrimSpace(content)
		if query == "" {
			return m, nil
		}
		m.AddToHistory(query)

		// Check if this is a slash command
		if msg := ParseSlashCommand(query); msg.Command != cmdInvalid {
			// Clear editor for slash commands (they're consumed, not kept as SQL)
			if msg.Command != cmdHelp {
				m.textarea.Reset()
				m.scrollOffset = 0
			}
			return m, func() tea.Msg {
				return msg
			}
		}

		return m, func() tea.Msg {
			return ExecuteQueryMsg{Query: query}
		}

	case "tab":
		// Insert 4 spaces for indentation
		m.textarea.InsertString("    ")
		m.updateScrollPosition(m.textarea.Line())
		return m, nil

	case "ctrl+u":
		// Clear editor (line delete in vim)
		m.textarea.Reset()
		m.scrollOffset = 0
		return m, nil
	}

	// Default: delegate to textarea
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	m.updateScrollPosition(m.textarea.Line())
	return m, cmd
}

// handleCommandModeKey processes keys in command mode (vim-like).
func (m SQLEditorModel) handleCommandModeKey(msg tea.KeyMsg) (SQLEditorModel, tea.Cmd) {
	switch msg.String() {
	case "i":
		// Switch to insert mode at current position
		m.mode = editorInsert
		return m, nil

	case "I":
		// Insert mode at the beginning of the line
		m.textarea.CursorStart()
		m.mode = editorInsert
		return m, nil

	case "A":
		// Insert mode at the end of the line
		m.textarea.CursorEnd()
		m.mode = editorInsert
		return m, nil

	case "o":
		// Insert new line below and enter insert mode
		m.textarea.InsertString("\n")
		m.mode = editorInsert
		m.updateScrollPosition(m.textarea.Line())
		return m, nil

	case "O":
		// Insert new line above and enter insert mode
		m.textarea.CursorStart()
		m.textarea.InsertString("\n")
		m.textarea.CursorUp()
		m.mode = editorInsert
		m.updateScrollPosition(m.textarea.Line())
		return m, nil

	case "j", "down":
		m.textarea.CursorDown()
		m.updateScrollPosition(m.textarea.Line())
		return m, nil

	case "k", "up":
		m.textarea.CursorUp()
		m.updateScrollPosition(m.textarea.Line())
		return m, nil

	case "h", "left":
		m.textarea.Update(tea.KeyMsg{Type: 'h'}) // let textarea handle
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

	case "l", "right":
		m.textarea.Update(tea.KeyMsg{Type: 'l'})
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd

	case "0", "home":
		m.textarea.CursorStart()
		return m, nil

	case "$", "end":
		m.textarea.CursorEnd()
		return m, nil

	case "G":
		// Go to last line
		m.textarea.CursorEnd()
		// Then go to the very bottom
		for i := 0; i < m.textarea.LineCount(); i++ {
			m.textarea.CursorDown()
		}
		m.updateScrollPosition(m.textarea.Line())
		return m, nil

	case "g":
		// gg: go to first line
		for m.textarea.Line() > 0 {
			m.textarea.CursorUp()
		}
		m.scrollOffset = 0
		return m, nil

	case "x":
		// Delete character under cursor
		m.textarea.InsertString("")
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(
			tea.KeyMsg{Type: tea.KeyDelete},
		)
		return m, cmd

	case "dd":
		// Delete current line
		currentLine := m.textarea.Line()
		m.textarea.CursorStart()
		// Delete to end of line, then delete newline
		for i := 0; i < 100 && m.textarea.Line() == currentLine; i++ {
			m.textarea, _ = m.textarea.Update(
				tea.KeyMsg{Type: tea.KeyDelete},
			)
		}
		return m, nil

	case "u":
		// Undo is not available in textarea, but we keep it no-op
		return m, nil

	case "ctrl+enter":
		// Execute query or slash command even in command mode
		content := m.textarea.Value()
		query := strings.TrimSpace(content)
		if query == "" {
			return m, nil
		}
		m.AddToHistory(query)

		// Check if this is a slash command
		if msg := ParseSlashCommand(query); msg.Command != cmdInvalid {
			if msg.Command != cmdHelp {
				m.textarea.Reset()
				m.scrollOffset = 0
			}
			return m, func() tea.Msg {
				return msg
			}
		}

		return m, func() tea.Msg {
			return ExecuteQueryMsg{Query: query}
		}

	case "enter":
		// In command mode, Enter switches to insert mode and adds newline
		m.mode = editorInsert
		m.textarea.InsertString("\n")
		m.updateScrollPosition(m.textarea.Line())
		return m, nil

	case "esc":
		// Already in command mode — no-op
		return m, nil
	}

	return m, nil
}

// View renders the editor with SQL syntax highlighting.
func (m SQLEditorModel) View() string {
	if !m.dirty && m.cachedView != "" {
		return m.cachedView
	}

	var b strings.Builder

	// Get content and cursor position
	content := m.textarea.Value()
	cursorRow := m.textarea.Line()
	lineInfo := m.textarea.LineInfo()
	cursorCol := lineInfo.ColumnOffset

	// Render highlighted SQL content
	contentWidth := m.viewport.Width
	if contentWidth < 10 {
		contentWidth = 10
	}
	highlighted := highlightSQL(content, cursorRow, cursorCol, true, contentWidth)

	// Apply stored scroll offset before viewport.Ready
	m.viewport.YOffset = m.scrollOffset

	// Set viewport content and render
	m.viewport.SetContent(highlighted)
	vpView := m.viewport.View()
	b.WriteString(vpView)

	// If content is empty, show placeholder
	if content == "" && !m.focused {
		_ = b.WriteByte('\n')
		placeholderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")).
			Italic(true)
		b.WriteString(placeholderStyle.Render("  Enter SQL here..."))
	}

	// Mode indicator line
	modeStr := m.renderModeIndicator()
	b.WriteString("\n")
	b.WriteString(modeStr)

	result := b.String()
	m.cachedView = result
	m.dirty = false
	return result
}

// updateScrollPosition adjusts the scroll offset so the cursor line
// stays visible in the viewport.
func (m *SQLEditorModel) updateScrollPosition(cursorRow int) {
	if m.viewport.Height <= 0 {
		return
	}
	visibleBottom := m.scrollOffset + m.viewport.Height - 1
	visibleTop := m.scrollOffset

	// Target zone: cursor in top 2/3s of viewport
	targetZoneBottom := m.scrollOffset + (m.viewport.Height * 2 / 3)
	if targetZoneBottom < 1 {
		targetZoneBottom = 1
	}

	if cursorRow < visibleTop {
		m.scrollOffset = cursorRow
	} else if cursorRow > visibleBottom-1 {
		m.scrollOffset = cursorRow - m.viewport.Height + 2
	} else if cursorRow > targetZoneBottom {
		m.scrollOffset = cursorRow - m.viewport.Height*2/3 + 1
	}

	// Clamp
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// renderModeIndicator shows the current mode (INSERT/COMMAND).
func (m SQLEditorModel) renderModeIndicator() string {
	var modeText string
	var color string

	switch m.mode {
	case editorInsert:
		modeText = " INSERT "
		color = "#A6E3A1" // green
	case editorCommand:
		modeText = " COMMAND "
		color = "#F9E2AF" // yellow
	}

	mode := m.modeIndicator.
		Background(lipgloss.Color(color)).
		Foreground(lipgloss.Color("#1E1E2E")).
		Render(modeText)

	// Show hints based on mode
	var hints string
	switch m.mode {
	case editorInsert:
		hints = " Ctrl+Enter: run/slash  |  /explain, /suggest, /optimize  |  Esc: cmd"
	case editorCommand:
		hints = " i: insert  |  j/k: move  |  dd: delete line  |  Ctrl+Enter: run"
	}

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7086")).
		Padding(0, 1)

	return mode + hintStyle.Render(hints)
}

// HistoryNavUp navigates backward through query history.
func (m *SQLEditorModel) HistoryNavUp() {
	if len(m.history) == 0 {
		return
	}
	if m.historyPos == -1 {
		// Save current content before browsing
		m.historyPos = len(m.history) - 1
	} else if m.historyPos > 0 {
		m.historyPos--
	}
	m.textarea.SetValue(m.history[m.historyPos])
	m.textarea.SetCursor(len([]rune(m.history[m.historyPos])))
	m.dirty = true
}

// HistoryNavDown navigates forward through query history.
func (m *SQLEditorModel) HistoryNavDown() {
	if m.historyPos == -1 || len(m.history) == 0 {
		return
	}
	m.historyPos++
	if m.historyPos >= len(m.history) {
		m.historyPos = -1
		m.textarea.SetValue("")
	} else {
		m.textarea.SetValue(m.history[m.historyPos])
		m.textarea.SetCursor(len([]rune(m.history[m.historyPos])))
	}
	m.dirty = true
}

// HistoryLen returns the number of entries in query history.
func (m SQLEditorModel) HistoryLen() int {
	return len(m.history)
}
