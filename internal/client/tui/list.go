package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	gophkeeper "github.com/liebeSonne/gophkeeper/pkg/client/gophkeeper"
)

type listTab int

const (
	tabData listTab = iota
	tabFiles
)

type ListItem struct {
	ID      uuid.UUID
	Title   string
	Type    string
	Size    string
	Status  string
	Created time.Time
	IsFile  bool
}

type ListModel struct {
	items      []ListItem
	selected   int
	page       int
	totalPages int
	total      int
	tab        listTab
	filterType string
	search     string
	width      int
	height     int
	loading    bool
	err        error
}

func NewListModel() ListModel {
	return ListModel{
		tab:    tabData,
		page:   1,
		width:  80,
		height: 24,
	}
}

func (m ListModel) Init() tea.Cmd {
	return nil
}

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.selected < len(m.items)-1 {
				m.selected++
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
			}
		case "g":
			m.page = 1
			m.selected = 0
		case "G":
			m.page = m.totalPages
		case "tab", "1":
			m.tab = tabData
			m.selected = 0
		case "2":
			m.tab = tabFiles
			m.selected = 0
		case "t":
			m.cycleFilterType()
		}
	}

	return m, nil
}

func (m *ListModel) cycleFilterType() {
	types := []string{"", "LOGIN_PASSWORD", "BANK_CARD", "TEXT", "FILE"}
	for i, t := range types {
		if m.filterType == t {
			m.filterType = types[(i+1)%len(types)]
			return
		}
	}
}

func (m *ListModel) SetItems(items []ListItem, page, totalPages, total int) {
	m.items = items
	m.page = page
	m.totalPages = totalPages
	m.total = total
	m.loading = false
	if m.selected >= len(m.items) && len(m.items) > 0 {
		m.selected = len(m.items) - 1
	}
}

func (m *ListModel) SetLoading(loading bool) {
	m.loading = loading
}

func (m *ListModel) SetError(err error) {
	m.err = err
	m.loading = false
}

func (m ListModel) GetSelectedItem() *ListItem {
	if m.selected < 0 || m.selected >= len(m.items) {
		return nil
	}
	return &m.items[m.selected]
}

func (m ListModel) GetTab() int {
	return int(m.tab)
}

func (m ListModel) GetPage() int {
	return m.page
}

func (m ListModel) GetFilterType() string {
	return m.filterType
}

func (m ListModel) GetSearch() string {
	return m.search
}

func (m *ListModel) SetSearch(s string) {
	m.search = s
}

func (m ListModel) View() string {
	var sb strings.Builder

	sb.WriteString(m.renderTabs())
	sb.WriteString("\n")

	if m.loading {
		sb.WriteString(styleFaint.Render("  Loading...\n"))
		return sb.String()
	}

	if m.err != nil {
		sb.WriteString(styleError.Render("  Error: " + m.err.Error() + "\n"))
		return sb.String()
	}

	sb.WriteString(m.renderFilter())
	sb.WriteString("\n")

	if len(m.items) == 0 {
		sb.WriteString(styleFaint.Render("  No items found\n"))
		sb.WriteString("\n")
		sb.WriteString(m.renderPagination())
		return sb.String()
	}

	sb.WriteString(m.renderItems())
	sb.WriteString("\n")
	sb.WriteString(m.renderPagination())

	return sb.String()
}

func (m ListModel) renderTabs() string {
	dataTab := tabStyle(m.tab == tabData).Render(" Data ")
	filesTab := tabStyle(m.tab == tabFiles).Render(" Files ")
	return "  " + dataTab + "  " + filesTab
}

func (m ListModel) renderFilter() string {
	if m.filterType == "" {
		return styleFaint.Render("  (t: filter by type)")
	}
	return styleActive.Render("  Filter: " + m.filterType + "  (t: change)")
}

func (m ListModel) renderItems() string {
	var sb strings.Builder

	header := fmt.Sprintf("  %-36s %-14s %s", "ID", "Type", "Created")
	if m.tab == tabFiles {
		header = fmt.Sprintf("  %-36s %-14s %-10s %s", "ID", "Name", "Size", "Status")
	}
	sb.WriteString(styleHeader.Render(header))
	sb.WriteString("\n")
	sb.WriteString(styleFaint.Render(strings.Repeat("-", 70)) + "\n")

	for i, item := range m.items {
		style := highlighted(i == m.selected)
		var line string

		if m.tab == tabData {
			line = fmt.Sprintf("  %-36s %-14s %s",
				truncateID(item.ID.String()),
				item.Type,
				item.Created.Format("2006-01-02"))
		} else {
			line = fmt.Sprintf("  %-36s %-14s %-10s %s",
				truncateID(item.ID.String()),
				item.Title,
				item.Size,
				item.Status)
		}

		sb.WriteString(style.Render(line))
		sb.WriteString("\n")
	}

	return sb.String()
}

func (m ListModel) renderPagination() string {
	return styleFaint.Render(fmt.Sprintf("  Page %d/%d (total: %d)  g/G: first/last",
		m.page, m.totalPages, m.total))
}

func (m ListModel) ToDataInfo() *gophkeeper.DataInfo {
	item := m.GetSelectedItem()
	if item == nil {
		return nil
	}
	return nil
}

func truncateID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}
