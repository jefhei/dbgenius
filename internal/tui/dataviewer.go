package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jefhei/dbgenius/internal/db"
)

// rowsPerPage is the default number of rows to show per page.
const rowsPerPage = 50

// DataViewerState represents the current state of the data viewer.
type DataViewerState int

const (
	viewerIdle DataViewerState = iota
	viewerLoading
	viewerLoaded
	viewerError
)

// TableSelectedMsg is sent when a table is selected in the schema tree.
type TableSelectedMsg struct {
	Schema string
	Table  string
}

// tableDataLoadedMsg is sent when table data has been loaded.
type tableDataLoadedMsg struct {
	columns []string
	rows    [][]*string
	total   int64
	schema  string
	table   string
}

// tableDataErrorMsg is sent when table data loading fails.
type tableDataErrorMsg struct {
	err    error
	schema string
	table  string
}

// DataViewerModel provides a paginated view of table data.
type DataViewerModel struct {
	state DataViewerState
	db    *db.IntrospectedBackend

	// Current table
	schema  string
	table   string
	columns []string

	// Pagination
	rows        [][]*string
	currentPage int
	totalRows   int64
	pageSize    int

	// Viewport for scrolling
	viewport viewport.Model

	// Styles
	headerStyle     lipgloss.Style
	cellStyle       lipgloss.Style
	nullStyle       lipgloss.Style
	helpStyle       lipgloss.Style
	titleStyle      lipgloss.Style
	loadingStyle    lipgloss.Style
	errorStyle      lipgloss.Style
	paginationStyle lipgloss.Style
	separatorStyle  lipgloss.Style

	width  int
	height int

	// Error message
	errMsg string
}

// NewDataViewerModel creates a new data viewer model.
func NewDataViewerModel() DataViewerModel {
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle()

	return DataViewerModel{
		state:       viewerIdle,
		pageSize:    rowsPerPage,
		currentPage: 0,
		viewport:    vp,

		headerStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#89B4FA")).
			Padding(0, 1).
			Background(lipgloss.Color("#313244")),
		cellStyle: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("#CDD6F4")),
		nullStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")).
			Italic(true).
			Padding(0, 1),
		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086")),
		titleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#A6E3A1")),
		loadingStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9E2AF")),
		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F38BA8")),
		paginationStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6ADC8")).
			Padding(0, 1).
			Background(lipgloss.Color("#1E1E2E")),
		separatorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#45475A")),
	}
}

// SetSize updates the viewer dimensions.
func (m *DataViewerModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	viewportH := height - 2 // Reserve for pagination bar
	if viewportH < 1 {
		viewportH = 1
	}
	m.viewport.Width = width
	m.viewport.Height = viewportH
}

// SelectTable tells the data viewer to load data for a table.
func (m *DataViewerModel) SelectTable(database *db.IntrospectedBackend, schema, table string) tea.Cmd {
	m.db = database
	m.schema = schema
	m.table = table
	m.currentPage = 0
	m.state = viewerLoading
	m.errMsg = ""
	return loadTableDataCmd(database, schema, table, 0, m.pageSize)
}

// Init implements tea.Model.
func (m DataViewerModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and user input for the data viewer.
func (m DataViewerModel) Update(msg tea.Msg) (DataViewerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tableDataLoadedMsg:
		m.state = viewerLoaded
		m.columns = msg.columns
		m.rows = msg.rows
		m.totalRows = msg.total
		// Reset viewport scroll position
		m.viewport.GotoTop()
		return m, nil

	case tableDataErrorMsg:
		m.state = viewerError
		m.errMsg = db.FriendlyError(msg.err)
		return m, nil

	case queryCancelledMsg:
		m.state = viewerIdle
		m.errMsg = ""
		return m, nil

	case tea.KeyMsg:
		if m.state != viewerLoaded {
			return m, nil
		}

		switch msg.String() {
		case "pgdown", "right":
			if m.currentPage < m.maxPage() {
				m.currentPage++
				m.state = viewerLoading
				return m, loadTableDataCmd(m.db, m.schema, m.table, m.currentPage, m.pageSize)
			}
		case "pgup", "left":
			if m.currentPage > 0 {
				m.currentPage--
				m.state = viewerLoading
				return m, loadTableDataCmd(m.db, m.schema, m.table, m.currentPage, m.pageSize)
			}
		case "home":
			if m.currentPage > 0 {
				m.currentPage = 0
				m.state = viewerLoading
				return m, loadTableDataCmd(m.db, m.schema, m.table, 0, m.pageSize)
			}
		case "end":
			maxPg := m.maxPage()
			if m.currentPage < maxPg {
				m.currentPage = maxPg
				m.state = viewerLoading
				return m, loadTableDataCmd(m.db, m.schema, m.table, m.currentPage, m.pageSize)
			}
		}
	}

	// Delegate to viewport for scrolling within results
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	return m, vpCmd
}

// maxPage returns the maximum page index (0-based).
func (m DataViewerModel) maxPage() int {
	if m.totalRows <= 0 {
		return 0
	}
	return int(m.totalRows-1) / m.pageSize
}

// View renders the data viewer.
func (m DataViewerModel) View() string {
	switch m.state {
	case viewerIdle:
		return m.helpStyle.Render("\n  Select a table in the schema tree\n  to view its data here.")
	case viewerLoading:
		return m.loadingStyle.Render("\n  ⏳ Loading data...")
	case viewerError:
		return fmt.Sprintf("%s\n%s",
			m.errorStyle.Render("  ✗ Error loading data:"),
			"  "+m.errMsg)
	case viewerLoaded:
		return m.renderTable()
	}
	return ""
}

// renderTable renders the paginated table view.
func (m DataViewerModel) renderTable() string {
	// Build content string
	var b strings.Builder

	// Title
	title := m.titleStyle.Render(fmt.Sprintf(" 📊 %s.%s", m.schema, m.table))
	b.WriteString(title)
	b.WriteString("\n")

	if len(m.rows) == 0 {
		b.WriteString(m.helpStyle.Render("  No data in this table.") + "\n")
		return b.String()
	}

	// Calculate column widths
	colWidths := m.calcColumnWidths()

	// Render header row
	for i, col := range m.columns {
		displayName := truncateDisplay(col, colWidths[i])
		header := m.headerStyle.Width(colWidths[i]).Render(displayName)
		b.WriteString(header)
		if i < len(m.columns)-1 {
			b.WriteString(" ")
		}
	}
	b.WriteString("\n")

	// Render separator
	for i := range m.columns {
		sep := strings.Repeat("─", colWidths[i])
		b.WriteString(m.separatorStyle.Render(sep))
		if i < len(m.columns)-1 {
			b.WriteString(" ")
		}
	}
	b.WriteString("\n")

	// Render data rows
	for _, row := range m.rows {
		for i, cell := range row {
			if i >= len(colWidths) {
				break
			}
			var cellStr string
			if cell == nil {
				cellStr = m.nullStyle.Width(colWidths[i]).Render("NULL")
			} else {
				cellStr = m.cellStyle.Width(colWidths[i]).Render(truncateDisplay(*cell, colWidths[i]))
			}
			b.WriteString(cellStr)
			if i < len(m.columns)-1 {
				b.WriteString(" ")
			}
		}
		b.WriteString("\n")
	}

	// Set viewport content
	m.viewport.SetContent(b.String())

	// Render viewport content with pagination
	viewportView := m.viewport.View()
	paginationBar := m.renderPagination()

	return viewportView + "\n" + paginationBar
}

// calcColumnWidths computes the display width for each column.
func (m DataViewerModel) calcColumnWidths() []int {
	colWidths := make([]int, len(m.columns))
	for i, col := range m.columns {
		colWidths[i] = len(col)
	}
	for _, row := range m.rows {
		for i, cell := range row {
			cellLen := 4 // "NULL"
			if cell != nil {
				cellLen = len(*cell)
			}
			if i < len(colWidths) && cellLen > colWidths[i] {
				colWidths[i] = cellLen
			}
		}
	}

	// Cap total width to viewer width (leave room for separators)
	numCols := len(m.columns)
	gapChars := 0
	if numCols > 1 {
		gapChars = numCols - 1
	}
	totalContentWidth := m.width - gapChars - 2 // -2 for safety margin
	if totalContentWidth > 0 {
		// Cap each column proportionally
		total := 0
		for _, w := range colWidths {
			total += w
		}
		if total > totalContentWidth {
			// Scale down proportionally
			for i := range colWidths {
				colWidths[i] = colWidths[i] * totalContentWidth / total
			}
			// Distribute remaining width
			remaining := totalContentWidth
			for _, w := range colWidths {
				remaining -= w
			}
			if remaining > 0 {
				colWidths[0] += remaining
			}
		}
	}

	// Enforce minimum width
	for i := range colWidths {
		if colWidths[i] < 3 {
			colWidths[i] = 3
		}
	}

	return colWidths
}

// renderPagination shows page info and navigation hints.
func (m DataViewerModel) renderPagination() string {
	firstRow := m.currentPage*m.pageSize + 1
	lastRow := firstRow + len(m.rows) - 1
	totalPages := m.maxPage() + 1

	info := fmt.Sprintf(" Rows %d-%d of %d | Page %d/%d ",
		firstRow, lastRow, m.totalRows, m.currentPage+1, totalPages)

	nav := " ←/→ pgup/pgdn navigate  Home/End first/last "

	padding := m.width - lipgloss.Width(m.paginationStyle.Render(info)) - lipgloss.Width(m.paginationStyle.Render(nav))
	if padding < 0 {
		padding = 0
	}
	spacer := strings.Repeat(" ", padding)

	return m.paginationStyle.Render(info + spacer + nav)
}

// truncateDisplay truncates a string to fit within maxLen, adding ellipsis.
func truncateDisplay(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 2 {
		return s[:maxLen]
	}
	return s[:maxLen-1] + "…"
}

// loadTableDataCmd creates a command that asynchronously loads table data.
func loadTableDataCmd(database *db.IntrospectedBackend, schema, table string, page, pageSize int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get total row count
		total, err := database.GetTableRowCount(ctx, schema, table)
		if err != nil {
			return tableDataErrorMsg{err: err, schema: schema, table: table}
		}

		// Build and execute paginated query
		offset := page * pageSize
		var query string
		dbType := database.GetType()
		switch dbType {
		case "postgres":
			query = fmt.Sprintf("SELECT * FROM %s.%s LIMIT %d OFFSET %d",
				quoteIdent(schema), quoteIdent(table), pageSize, offset)
		case "sqlite":
			query = fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d",
				quoteIdent(table), pageSize, offset)
		default:
			query = fmt.Sprintf("SELECT * FROM %s LIMIT %d OFFSET %d",
				quoteIdent(table), pageSize, offset)
		}

		result, err := database.ExecuteQuery(ctx, query)
		if err != nil {
			return tableDataErrorMsg{err: err, schema: schema, table: table}
		}

		return tableDataLoadedMsg{
			columns: result.Columns,
			rows:    result.Rows,
			total:   total,
			schema:  schema,
			table:   table,
		}
	}
}

// quoteIdent quotes an identifier for SQL.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
