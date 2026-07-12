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

// copyCellMsg is sent when a cell content is copied to clipboard.
type copyCellMsg struct {
	content string
}

// DataViewerModel provides a paginated view of table data with
// horizontal scrolling, cell selection, and clipboard copy support.
type DataViewerModel struct {
	state DataViewerState
	db    *db.IntrospectedBackend

	// Current table / query
	schema  string
	table   string
	columns []string

	// Pagination
	rows        [][]*string
	currentPage int
	totalRows   int64
	pageSize    int

	// Viewport for vertical scrolling (data rows only, header is pinned)
	viewport viewport.Model

	// Horizontal scrolling
	hScrollOffset int

	// Cell cursor for selection and copy
	selectedRow int // row index within current page
	selectedCol int // column index

	// Styles
	headerStyle       lipgloss.Style
	cellStyle         lipgloss.Style
	nullStyle         lipgloss.Style
	helpStyle         lipgloss.Style
	titleStyle        lipgloss.Style
	loadingStyle      lipgloss.Style
	errorStyle        lipgloss.Style
	paginationStyle   lipgloss.Style
	separatorStyle    lipgloss.Style
	selectedCellStyle lipgloss.Style
	copiedStyle       lipgloss.Style

	width  int
	height int

	// Error message
	errMsg string

	// Feedback message (e.g., "Copied!")
	feedbackMsg   string
	feedbackTimer int // ticks remaining for feedback display
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
		selectedCellStyle: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("#1E1E2E")).
			Background(lipgloss.Color("#89B4FA")),
		copiedStyle: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("#1E1E2E")).
			Background(lipgloss.Color("#A6E3A1")),
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
	viewportH := height - 3 // Reserve for title + pagination bar
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
	m.hScrollOffset = 0
	m.selectedRow = 0
	m.selectedCol = 0
	m.feedbackMsg = ""
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
		m.hScrollOffset = 0
		m.selectedRow = 0
		m.selectedCol = 0
		m.feedbackMsg = ""
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

	case copyCellMsg:
		// Feedback for clipboard copy
		m.feedbackMsg = fmt.Sprintf("📋 Copied: %s", truncateDisplay(msg.content, 40))
		m.feedbackTimer = 2
		return m, nil

	case tea.KeyMsg:
		if m.state != viewerLoaded {
			return m, nil
		}

		switch msg.String() {
		case "pgdown":
			if m.currentPage < m.maxPage() {
				m.currentPage++
				m.hScrollOffset = 0
				m.selectedRow = 0
				m.state = viewerLoading
				return m, loadTableDataCmd(m.db, m.schema, m.table, m.currentPage, m.pageSize)
			}
		case "pgup":
			if m.currentPage > 0 {
				m.currentPage--
				m.hScrollOffset = 0
				m.selectedRow = 0
				m.state = viewerLoading
				return m, loadTableDataCmd(m.db, m.schema, m.table, m.currentPage, m.pageSize)
			}
		case "home":
			if m.currentPage > 0 {
				m.currentPage = 0
				m.hScrollOffset = 0
				m.selectedRow = 0
				m.state = viewerLoading
				return m, loadTableDataCmd(m.db, m.schema, m.table, 0, m.pageSize)
			}
		case "end":
			maxPg := m.maxPage()
			if m.currentPage < maxPg {
				m.currentPage = maxPg
				m.hScrollOffset = 0
				m.selectedRow = 0
				m.state = viewerLoading
				return m, loadTableDataCmd(m.db, m.schema, m.table, m.currentPage, m.pageSize)
			}
		case "right":
			// Horizontal scroll right
			maxCols := len(m.columns)
			if m.selectedCol < maxCols-1 {
				m.selectedCol++
			}
			m.clampHScroll()
		case "left":
			// Horizontal scroll left
			if m.selectedCol > 0 {
				m.selectedCol--
			}
			m.clampHScroll()
		case "down":
			if m.selectedRow < len(m.rows)-1 {
				m.selectedRow++
			}
			// Also scroll viewport to keep cursor visible
			m.ensureCursorVisible()
		case "up":
			if m.selectedRow > 0 {
				m.selectedRow--
			}
			m.ensureCursorVisible()
		case "y", "Y":
			// Copy selected cell to clipboard
			if m.selectedRow < len(m.rows) && m.selectedCol < len(m.columns) {
				cell := m.rows[m.selectedRow][m.selectedCol]
				content := ""
				if cell != nil {
					content = *cell
				}
				// Return a command that sends back a copy confirmation
				return m, func() tea.Msg {
					return copyCellMsg{content: content}
				}
			}
		}

		// Tick down feedback timer
		if m.feedbackTimer > 0 {
			m.feedbackTimer--
			if m.feedbackTimer <= 0 {
				m.feedbackMsg = ""
			}
		}

		return m, nil
	}

	// Delegate to viewport for vertical scrolling
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	return m, vpCmd
}

// clampHScroll ensures the horizontal scroll offset keeps the selected column visible.
func (m *DataViewerModel) clampHScroll() {
	if len(m.columns) == 0 {
		m.hScrollOffset = 0
		return
	}

	// Calculate how many columns we can show
	visibleCols := m.visibleColumnCount()
	if visibleCols <= 0 {
		return
	}

	// If selected col is beyond the right edge, scroll right
	if m.selectedCol >= m.hScrollOffset+visibleCols {
		m.hScrollOffset = m.selectedCol - visibleCols + 1
	}

	// If selected col is before the left edge, scroll left
	if m.selectedCol < m.hScrollOffset {
		m.hScrollOffset = m.selectedCol
	}

	// Clamp hScrollOffset
	if m.hScrollOffset > len(m.columns)-visibleCols {
		m.hScrollOffset = len(m.columns) - visibleCols
	}
	if m.hScrollOffset < 0 {
		m.hScrollOffset = 0
	}
}

// ensureCursorVisible scrolls the viewport vertically so the selected row is visible.
func (m *DataViewerModel) ensureCursorVisible() {
	if m.viewport.Height <= 0 {
		return
	}

	// Each data row takes 1 line
	cursorLine := m.selectedRow
	top := m.viewport.YOffset
	bottom := top + m.viewport.Height - 1

	if cursorLine < top {
		m.viewport.YOffset = cursorLine
	} else if cursorLine >= bottom {
		m.viewport.YOffset = cursorLine - m.viewport.Height + 2
	}

	// Clamp
	if m.viewport.YOffset < 0 {
		m.viewport.YOffset = 0
	}
}

// visibleColumnCount returns how many columns can fit in the available width.
func (m DataViewerModel) visibleColumnCount() int {
	colWidths := m.calcColumnWidths()
	total := 0
	count := 0
	gapChars := 1 // one space between columns
	for i := m.hScrollOffset; i < len(colWidths); i++ {
		needed := colWidths[i] + 2 // padding
		if count > 0 {
			needed += gapChars
		}
		if total+needed > m.width {
			break
		}
		total += needed
		count++
	}
	if count == 0 && len(colWidths) > 0 {
		return 1 // at least show one column
	}
	return count
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

// renderTable renders the paginated table view with pinned header and horizontal scrolling.
func (m DataViewerModel) renderTable() string {
	// Build content with pinned header
	var b strings.Builder

	// Title line
	if m.schema != "" && m.table != "" {
		title := m.titleStyle.Render(fmt.Sprintf(" 📊 %s.%s", m.schema, m.table))
		b.WriteString(title)
	} else {
		title := m.titleStyle.Render(" 📊 Query Result")
		b.WriteString(title)
	}
	b.WriteString("\n")

	if len(m.rows) == 0 {
		b.WriteString(m.helpStyle.Render("  No data.") + "\n")
		return b.String()
	}

	// Calculate column widths (unscaled for scrolling)
	colWidths := m.calcColumnWidths()

	// Determine which columns are visible
	visibleCols := m.visibleColumnCount()
	if visibleCols > len(m.columns)-m.hScrollOffset {
		visibleCols = len(m.columns) - m.hScrollOffset
	}
	endCol := m.hScrollOffset + visibleCols
	if endCol > len(m.columns) {
		endCol = len(m.columns)
	}

	// Calculate actual rendered column widths within available space
	renderedWidths := m.calcRenderedWidths(colWidths, m.hScrollOffset, endCol)

	// Render pinned header row
	headerRow := m.renderHeaderRow(m.columns[m.hScrollOffset:endCol], renderedWidths)
	b.WriteString(headerRow)
	b.WriteString("\n")

	// Render separator
	sepRow := m.renderSeparatorRow(renderedWidths)
	b.WriteString(sepRow)
	b.WriteString("\n")

	// Render data rows (with cell selection highlighting)
	for rowIdx, row := range m.rows {
		for colOffset, cell := range row[m.hScrollOffset:endCol] {
			colIdx := m.hScrollOffset + colOffset
			cellStr := m.renderCell(cell, renderedWidths[colOffset], rowIdx == m.selectedRow && colIdx == m.selectedCol)
			b.WriteString(cellStr)
			if colOffset < len(renderedWidths)-1 {
				b.WriteString(" ")
			}
		}
		b.WriteString("\n")
	}

	// Set viewport content (data rows only — header is separate)
	m.viewport.SetContent(b.String())

	// Render viewport
	viewportView := m.viewport.View()

	// Build feedback bar (copy confirmation, scroll indicator)
	feedbackBar := m.renderFeedbackBar(colWidths, m.hScrollOffset, endCol)

	// Build pagination bar
	paginationBar := m.renderPagination()

	return viewportView + "\n" + feedbackBar + paginationBar
}

// renderHeaderRow renders the table header for visible columns.
func (m DataViewerModel) renderHeaderRow(columns []string, widths []int) string {
	var b strings.Builder
	for i, col := range columns {
		displayName := truncateDisplay(col, widths[i])
		header := m.headerStyle.Width(widths[i]).Render(displayName)
		b.WriteString(header)
		if i < len(columns)-1 {
			b.WriteString(" ")
		}
	}
	return b.String()
}

// renderSeparatorRow renders the separator line.
func (m DataViewerModel) renderSeparatorRow(widths []int) string {
	var b strings.Builder
	for i, w := range widths {
		sep := strings.Repeat("─", w)
		b.WriteString(m.separatorStyle.Render(sep))
		if i < len(widths)-1 {
			b.WriteString(" ")
		}
	}
	return b.String()
}

// renderCell renders a single cell with optional selection highlighting.
func (m DataViewerModel) renderCell(cell *string, width int, selected bool) string {
	var cellStr string
	cellContent := "NULL"
	if cell != nil {
		cellContent = *cell
	}

	if selected {
		// Selected cell gets highlighted background
		cellStr = m.selectedCellStyle.Width(width).Render(truncateDisplay(cellContent, width))
	} else if cell == nil {
		cellStr = m.nullStyle.Width(width).Render("NULL")
	} else {
		cellStr = m.cellStyle.Width(width).Render(truncateDisplay(cellContent, width))
	}
	return cellStr
}

// renderFeedbackBar shows copy feedback and horizontal scroll indicator.
func (m DataViewerModel) renderFeedbackBar(colWidths []int, hOffset, endCol int) string {
	var parts []string

	// Copy feedback
	if m.feedbackMsg != "" {
		parts = append(parts, m.copiedStyle.Render(m.feedbackMsg))
	}

	// Horizontal scroll indicator
	totalCols := len(m.columns)
	if totalCols > 0 {
		scrollInfo := fmt.Sprintf(" Cols %d-%d of %d ", hOffset+1, endCol, totalCols)
		parts = append(parts, m.paginationStyle.Render(scrollInfo))
	}

	if len(parts) == 0 {
		return ""
	}

	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p)
	}
	b.WriteString("\n")
	return b.String()
}

// renderPagination shows page info and navigation hints.
func (m DataViewerModel) renderPagination() string {
	firstRow := m.currentPage*m.pageSize + 1
	lastRow := firstRow + len(m.rows) - 1
	totalPages := m.maxPage() + 1

	info := fmt.Sprintf(" Rows %d-%d of %d | Page %d/%d ",
		firstRow, lastRow, m.totalRows, m.currentPage+1, totalPages)

	nav := " pgup/pgdn: page  ←/→: hscroll  ↑/↓: cursor  y: copy "

	padding := m.width - lipgloss.Width(m.paginationStyle.Render(info)) - lipgloss.Width(m.paginationStyle.Render(nav))
	if padding < 0 {
		padding = 0
	}
	spacer := strings.Repeat(" ", padding)

	return m.paginationStyle.Render(info + spacer + nav)
}

// calcColumnWidths computes the natural display width for each column
// (before horizontal scroll limiting).
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

	// Cap each column to a reasonable max to prevent layout issues
	maxColWidth := 60
	for i := range colWidths {
		if colWidths[i] > maxColWidth {
			colWidths[i] = maxColWidth
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

// calcRenderedWidths adjusts column widths to fit the available viewport width.
func (m DataViewerModel) calcRenderedWidths(colWidths []int, startCol, endCol int) []int {
	visible := endCol - startCol
	if visible <= 0 {
		return nil
	}

	widths := make([]int, visible)
	copy(widths, colWidths[startCol:endCol])

	// Account for separator gaps between columns
	numGaps := 0
	if visible > 1 {
		numGaps = visible - 1
	}
	gapChars := 1 // one space between columns
	paddingChars := 2 * visible // left/right padding per column (1 each side)
	totalReserved := numGaps*gapChars + paddingChars
	availableWidth := m.width - totalReserved
	if availableWidth < 10 {
		availableWidth = 10
	}

	// Sum natural widths
	total := 0
	for _, w := range widths {
		total += w
	}

	// Scale down if too wide
	if total > availableWidth {
		for i := range widths {
			widths[i] = widths[i] * availableWidth / total
		}
		// Distribute remainder
		used := 0
		for _, w := range widths {
			used += w
		}
		remainder := availableWidth - used
		for i := range widths {
			if remainder <= 0 {
				break
			}
			widths[i]++
			remainder--
		}
	}

	// Ensure minimum width
	for i := range widths {
		if widths[i] < 3 {
			widths[i] = 3
		}
	}

	return widths
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
