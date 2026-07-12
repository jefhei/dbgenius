package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1E1E2E")).
			Foreground(lipgloss.Color("#A6ADC8")).
			Padding(0, 1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086"))

	focusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#7C3AED"))

	unfocusedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#363A4F"))

	helpOverlayStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color("#7C3AED")).
				Padding(1, 2).
				Background(lipgloss.Color("#1E1E2E"))

	helpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F5C2E7")).
			Padding(0, 0)

	helpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#89B4FA"))

	helpDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CDD6F4"))

	helpSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#A6E3A1"))
)

// View renders the complete TUI.
func (m RootModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	if m.showHelp {
		return m.renderHelpOverlay()
	}

	// Build panels
	schemaPanel := m.renderSchemaPanel()
	editorPanel := m.renderEditorPanel()
	resultsPanel := m.renderResultsPanel()
	statusBar := m.renderStatusBar()

	// Layout: Left (schema) | Right (editor top + results bottom)
	leftWidth := 30
	if leftWidth > m.width/4 {
		leftWidth = m.width / 4
	}
	rightWidth := m.width - leftWidth - 3

	rightTopHeight := m.height / 3
	rightBottomHeight := m.height - rightTopHeight - 2

	schemaBorder := unfocusedBorderStyle
	editorBorder := unfocusedBorderStyle
	resultsBorder := unfocusedBorderStyle

	switch m.focusedPanel {
	case panelSchemaTree:
		schemaBorder = focusedBorderStyle
	case panelEditor:
		editorBorder = focusedBorderStyle
	case panelResults:
		resultsBorder = focusedBorderStyle
	}

	if leftWidth < 10 {
		leftWidth = 10
	}
	if rightWidth < 20 {
		rightWidth = 20
	}
	if rightTopHeight < 3 {
		rightTopHeight = 3
	}
	if rightBottomHeight < 3 {
		rightBottomHeight = 3
	}

	schemaView := schemaBorder.Width(leftWidth).Height(m.height - 2).Render(schemaPanel)
	editorView := editorBorder.Width(rightWidth).Height(rightTopHeight).Render(editorPanel)
	resultsView := resultsBorder.Width(rightWidth).Height(rightBottomHeight).Render(resultsPanel)

	rightSide := lipgloss.JoinVertical(lipgloss.Top, editorView, resultsView)
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, schemaView, " ", rightSide)

	return lipgloss.JoinVertical(lipgloss.Left, mainView, statusBar)
}

func (m RootModel) renderSchemaPanel() string {
	title := titleStyle.Render("📂 Schemas")
	treeContent := m.schemaTree.View()
	helpLine := helpStyle.Render("\n  ↑↓ navigate  → expand  ← collapse")
	return fmt.Sprintf("%s\n%s%s", title, treeContent, helpLine)
}

func (m RootModel) renderEditorPanel() string {
	title := titleStyle.Render("✏️ SQL Editor")
	editorContent := m.sqlEditor.View()
	return fmt.Sprintf("%s\n%s", title, editorContent)
}

func (m RootModel) renderResultsPanel() string {
	title := titleStyle.Render("📊 Results")
	content := m.dataViewer.View()
	if content == "" || (m.aiResponse != "" && m.dataViewer.state != viewerLoading && m.dataViewer.state != viewerError) {
		// Show AI response if available
		if m.aiResponse != "" {
			content = m.renderAIResponse()
		} else if m.isExecuting {
			content = m.renderQueryLoading()
		} else {
			content = "  Run a query to see results"
		}
	}
	return fmt.Sprintf("%s\n%s", title, content)
}

// renderAIResponse displays the AI's response formatted for the TUI.
func (m RootModel) renderAIResponse() string {
	var b strings.Builder
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CDD6F4")).
		Padding(0, 1)
	b.WriteString(style.Render(m.aiResponse))
	return b.String()
}

// renderQueryLoading displays a loading indicator during query execution.
func (m RootModel) renderQueryLoading() string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F9E2AF")).
		Italic(true)
	return style.Render("\n  ⏳ Executing query...\n  Press Ctrl+C to cancel")
}

func (m RootModel) renderStatusBar() string {
	focusNames := map[panel]string{
		panelSchemaTree: "SCHEMA TREE",
		panelEditor:     "EDITOR",
		panelResults:    "RESULTS",
	}
	focusStr := focusNames[m.focusedPanel]

	// Connection status
	var dbInfo string
	var connStatus string
	if m.db != nil {
		dbType := m.db.GetType()
		dbInfo = fmt.Sprintf(" %s ", strings.ToUpper(dbType))
		connStatus = " ● "
	} else {
		dbInfo = " NO DB "
		connStatus = " ○ "
	}

	// Selected table info
	var tableInfo string
	if m.dataViewer.schema != "" && m.dataViewer.table != "" {
		tableInfo = fmt.Sprintf(" %s.%s ", m.dataViewer.schema, m.dataViewer.table)
	} else {
		tableInfo = ""
	}

	var execStatus string
	if m.isExecuting {
		execStatus = " ⏳ QUERY RUNNING... "
	} else {
		execStatus = ""
	}

	connStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A6E3A1")) // green for connected
	if m.db == nil {
		connStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F38BA8")) // red for disconnected
	}

	left := statusBarStyle.Render(fmt.Sprintf(" dbgenius |%s|%s %s%s%s ",
		dbInfo, connStyle.Render(connStatus), focusStr, tableInfo, execStatus))
	right := statusBarStyle.Render(" Ctrl+C: quit/cancel  |  Tab/Ctrl+W: switch  |  Esc: cmd  |  ?: help")

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	center := strings.Repeat(" ", gap)

	return lipgloss.JoinHorizontal(lipgloss.Bottom, left, center, right)
}

// renderHelpOverlay renders the help/cheatsheet overlay.
func (m RootModel) renderHelpOverlay() string {
	helpContent := m.buildHelpContent()
	overlay := helpOverlayStyle.
		Width(m.width - 4).
		Height(m.height - 2).
		Render(helpContent)
	return overlay
}

// buildHelpContent returns the help text shown when ? is pressed.
func (m RootModel) buildHelpContent() string {
	var b strings.Builder

	b.WriteString(helpTitleStyle.Render("📖 dbgenius Help"))
	b.WriteString("\n\n")

	// General
	b.WriteString(helpSectionStyle.Render("General"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("Ctrl+C / q"), helpDescStyle.Render("Quit dbgenius")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("Tab / Shift+Tab"), helpDescStyle.Render("Cycle focus between panels")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("Ctrl+W / Ctrl+Shift+Tab"), helpDescStyle.Render("Alternative focus switch")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("?"), helpDescStyle.Render("Toggle this help screen")))
	if m.aiClient != nil {
		statusIcon := "✅"
		if !m.ollamaAvailable {
			statusIcon = "❌"
		}
		b.WriteString(fmt.Sprintf("  %s  %s (%s)\n", statusIcon, helpDescStyle.Render("Ollama"), helpDescStyle.Render(m.aiClient.BaseURL())))
	}
	b.WriteString("\n")

	// Schema Tree
	b.WriteString(helpSectionStyle.Render("Schema Browser (left panel)"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("↑ / k"), helpDescStyle.Render("Move up")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("↓ / j"), helpDescStyle.Render("Move down")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("→ / Enter"), helpDescStyle.Render("Expand schema")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("←"), helpDescStyle.Render("Collapse schema")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("Enter (on table)"), helpDescStyle.Render("View table data")))
	b.WriteString("\n")

	// SQL Editor
	b.WriteString(helpSectionStyle.Render("SQL Editor (top-right panel)"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("Type"), helpDescStyle.Render("Enter SQL query (insert mode)")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("Ctrl+Enter"), helpDescStyle.Render("Execute query or slash command")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("/explain"), helpDescStyle.Render("Explain query with AI")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("/suggest <req>"), helpDescStyle.Render("Suggest query with AI")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("/optimize"), helpDescStyle.Render("Optimize query with AI")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("/help"), helpDescStyle.Render("Show slash commands")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("Esc"), helpDescStyle.Render("Switch to command mode")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("i"), helpDescStyle.Render("Switch to insert mode")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("Tab"), helpDescStyle.Render("Insert 4-space indent")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("Ctrl+U"), helpDescStyle.Render("Clear editor")))
	b.WriteString("\n")

	// Command Mode
	b.WriteString(helpSectionStyle.Render("Command Mode (vim-like)"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("j/k"), helpDescStyle.Render("Move cursor down/up")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("h/l"), helpDescStyle.Render("Move cursor left/right")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("0 / $"), helpDescStyle.Render("Go to start/end of line")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("dd"), helpDescStyle.Render("Delete current line")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("x"), helpDescStyle.Render("Delete character")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("o / O"), helpDescStyle.Render("Insert new line below/above")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("I / A"), helpDescStyle.Render("Insert at start/end of line")))
	b.WriteString("\n")

	// Results
	b.WriteString(helpSectionStyle.Render("Results Viewer (bottom-right panel)"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("PgUp / PgDn"), helpDescStyle.Render("Previous/next page")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("Home / End"), helpDescStyle.Render("First/last page")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("← / →"), helpDescStyle.Render("Horizontal scroll columns")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("↑ / ↓"), helpDescStyle.Render("Move cell cursor (select cell)")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", helpKeyStyle.Render("Y / y"), helpDescStyle.Render("Copy selected cell content")))

	return b.String()
}
