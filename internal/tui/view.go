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
)

// View renders the complete TUI.
func (m RootModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Build panels
	schemaPanel := m.renderSchemaPanel()
	editorPanel := m.renderEditorPanel()
	resultsPanel := m.renderResultsPanel()
	statusBar := m.renderStatusBar()

	// Layout: Left (schema) | Right (editor top + results bottom)
	// The layout adapts to available space
	leftWidth := 30
	if leftWidth > m.width/4 {
		leftWidth = m.width / 4
	}
	rightWidth := m.width - leftWidth - 3 // 3 for borders/gaps

	rightTopHeight := m.height / 3
	rightBottomHeight := m.height - rightTopHeight - 2 // 2 for status bar

	// Build panels with borders
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
	items := "  No databases connected\n\n  Connect to a database to\n  browse schemas and tables."
	return fmt.Sprintf("%s\n%s", title, items)
}

func (m RootModel) renderEditorPanel() string {
	title := titleStyle.Render("✏️ SQL Editor")
	placeholder := "  Enter SQL here...\n  Ctrl+Enter to execute"
	help := helpStyle.Render("\n  Tab: focus  |  ?: help")
	return fmt.Sprintf("%s\n%s%s", title, placeholder, help)
}

func (m RootModel) renderResultsPanel() string {
	title := titleStyle.Render("📊 Results")
	placeholder := "  Run a query to see results"
	return fmt.Sprintf("%s\n%s", title, placeholder)
}

func (m RootModel) renderStatusBar() string {
	focusNames := map[panel]string{
		panelSchemaTree: "SCHEMA TREE",
		panelEditor:     "EDITOR",
		panelResults:    "RESULTS",
	}
	focusStr := focusNames[m.focusedPanel]

	// Build status bar
	left := statusBarStyle.Render(fmt.Sprintf(" dbgenius | %s ", focusStr))
	right := statusBarStyle.Render(" Ctrl+C: quit  |  ?: help ")
	
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	center := strings.Repeat(" ", gap)

	return lipgloss.JoinHorizontal(lipgloss.Bottom, left, center, right)
}
