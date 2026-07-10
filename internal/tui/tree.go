package tui

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jefhei/dbgenius/internal/db"
)

// TreeNode represents a node in the schema/table tree.
type TreeNode struct {
	Label    string
	Icon     string
	Depth    int
	Expanded bool
	HasChildren bool
	// Reference data
	Schema string
	Table  string
	// Children loaded state
	childrenLoaded bool
}

// TreeModel implements a tree view for browsing database schemas and tables.
type TreeModel struct {
	viewport viewport.Model
	nodes    []TreeNode
	cursor   int
	width    int
	height   int

	// Connected database
	db          *db.IntrospectedBackend
	dbConnected bool
	dbType      string

	// Styles
	itemStyle        lipgloss.Style
	selectedStyle    lipgloss.Style
	dimStyle         lipgloss.Style
	iconStyle        lipgloss.Style
	indentWidth      int
}

// NewTreeModel creates a new tree view model.
func NewTreeModel() TreeModel {
	vp := viewport.New(30, 20)
	vp.Style = lipgloss.NewStyle()

	return TreeModel{
		viewport:     vp,
		nodes:        []TreeNode{},
		cursor:       0,
		indentWidth:  2,
		itemStyle: lipgloss.NewStyle().
				Foreground(lipgloss.Color("#CDD6F4")).
				Padding(0, 1),
		selectedStyle: lipgloss.NewStyle().
				Background(lipgloss.Color("#45475A")).
				Foreground(lipgloss.Color("#CDD6F4")).
				Padding(0, 1),
		dimStyle: lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6C7086")).
				Padding(0, 1),
		iconStyle: lipgloss.NewStyle().
				Foreground(lipgloss.Color("#89B4FA")),
	}
}

// SetDB connects the tree to a database for live browsing.
func (m *TreeModel) SetDB(database *db.IntrospectedBackend, dbType string) {
	m.db = database
	m.dbConnected = true
	m.dbType = dbType
}

// ClearDB disconnects the tree from the database.
func (m *TreeModel) ClearDB() {
	m.db = nil
	m.dbConnected = false
	m.nodes = nil
	m.cursor = 0
}

// Init initializes the tree model.
func (m TreeModel) Init() tea.Cmd {
	return nil
}

// SetSize updates the tree dimensions.
func (m *TreeModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
}

// Update handles tree events and returns commands.
func (m TreeModel) Update(msg tea.Msg) (TreeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
		case "down", "j":
			if m.cursor < len(m.nodes)-1 {
				m.cursor++
				m.ensureVisible()
			}
		case "right", "l", "enter":
			return m.handleEnter()
		case "left", "h":
			m.collapseCurrent()
		}
	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
	}

	return m, nil
}

// handleEnter expands/collapses schemas or selects tables.
func (m TreeModel) handleEnter() (TreeModel, tea.Cmd) {
	if m.cursor >= len(m.nodes) {
		return m, nil
	}

	node := &m.nodes[m.cursor]
	if node.HasChildren {
		if node.Expanded {
			m.collapseCurrent()
		} else {
			node.Expanded = true
		}
		return m, nil
	}

	// Table node selected — emit TableSelectedMsg
	if node.Schema != "" && node.Table != "" {
		return m, func() tea.Msg {
			return TableSelectedMsg{
				Schema: node.Schema,
				Table:  node.Table,
			}
		}
	}

	return m, nil
}

// View renders the tree.
func (m TreeModel) View() string {
	if !m.dbConnected {
		return m.dimStyle.Render("\n  No database connected.\n  Connect to browse schemas\n  and tables.")
	}

	if len(m.nodes) == 0 {
		return m.dimStyle.Render("\n  No tables found.")
	}

	var renderedLines []string
	for i, node := range m.nodes {
		line := m.renderNode(node, i == m.cursor)
		renderedLines = append(renderedLines, line)
	}

	// Build the tree content
	content := ""
	for _, line := range renderedLines {
		content += line + "\n"
	}

	m.viewport.SetContent(content)
	return m.viewport.View()
}

func (m TreeModel) renderNode(node TreeNode, selected bool) string {
	indent := ""
	for i := 0; i < node.Depth; i++ {
		indent += "  "
	}

	// Expand/collapse indicator
	var expander string
	if node.HasChildren {
		if node.Expanded {
			expander = "▼ "
		} else {
			expander = "▶ "
		}
	} else {
		expander = "  "
	}

	// Icon
	icon := m.iconStyle.Render(node.Icon + " ")

	label := node.Label
	if selected {
		return m.selectedStyle.Render(indent + expander + icon + label)
	}
	return m.itemStyle.Render(indent + expander + icon + label)
}

func (m *TreeModel) ensureVisible() {
	// Scroll to keep cursor visible
	m.viewport.GotoTop()
}

// LoadSchemas loads schemas from the connected database.
func (m *TreeModel) LoadSchemas(schemas []string) {
	m.nodes = []TreeNode{}
	for _, s := range schemas {
		m.nodes = append(m.nodes, TreeNode{
			Label:       s,
			Icon:        "📁",
			Depth:       0,
			Expanded:    false,
			HasChildren: true,
			Schema:      s,
			childrenLoaded: false,
		})
	}
	m.cursor = 0
}

// LoadTables adds tables to a schema node.
func (m *TreeModel) LoadTables(schemaIdx int, tables []db.TableInfo) {
	if schemaIdx >= len(m.nodes) {
		return
	}

	// Remove any existing child nodes of this schema
	m.removeChildren(schemaIdx)

	// Add table nodes after the schema node
	newNodes := make([]TreeNode, 0, len(m.nodes)+len(tables))
	for i, node := range m.nodes {
		newNodes = append(newNodes, node)
		if i == schemaIdx {
			for _, t := range tables {
				newNodes = append(newNodes, TreeNode{
					Label:       t.Name,
					Icon:        "📊",
					Depth:       1,
					Expanded:    false,
					HasChildren: false,
					Schema:      node.Schema,
					Table:       t.Name,
					childrenLoaded: false,
				})
			}
		}
	}
	m.nodes = newNodes

	// Mark schema as expanded
	if schemaIdx < len(m.nodes) {
		m.nodes[schemaIdx].Expanded = true
		m.nodes[schemaIdx].childrenLoaded = true
	}
}

func (m *TreeModel) collapseCurrent() {
	if m.cursor >= len(m.nodes) {
		return
	}

	node := &m.nodes[m.cursor]
	if !node.HasChildren {
		return
	}

	// Remove children
	m.removeChildren(m.cursor)
	node.Expanded = false
}

func (m *TreeModel) removeChildren(parentIdx int) {
	if parentIdx+1 >= len(m.nodes) {
		return
	}

	// Find where children end (next node with depth <= parent depth)
	parentDepth := m.nodes[parentIdx].Depth
	endIdx := parentIdx + 1
	for endIdx < len(m.nodes) && m.nodes[endIdx].Depth > parentDepth {
		endIdx++
	}

	if endIdx > parentIdx+1 {
		// Adjust cursor if needed
		if m.cursor > parentIdx {
			cursorOffset := m.cursor - parentIdx
			if cursorOffset < endIdx-parentIdx {
				m.cursor = parentIdx
			} else {
				m.cursor -= (endIdx - parentIdx - 1)
			}
		}

		m.nodes = append(m.nodes[:parentIdx+1], m.nodes[endIdx:]...)
	}
}

// SelectedNode returns the currently selected node info.
func (m TreeModel) SelectedNode() (string, string, bool) {
	if m.cursor >= len(m.nodes) {
		return "", "", false
	}
	node := m.nodes[m.cursor]
	return node.Schema, node.Table, !node.HasChildren
}

// GetCursor returns the current cursor position.
func (m TreeModel) GetCursor() int {
	return m.cursor
}

// GetNodes returns all nodes.
func (m TreeModel) GetNodes() []TreeNode {
	return m.nodes
}
