package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type SearchModel struct {
	query    string
	active   bool
	width    int
	height   int
	onSubmit func(string)
}

func NewSearchModel() SearchModel {
	return SearchModel{}
}

func (m SearchModel) Init() tea.Cmd {
	return nil
}

func (m SearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.active = false
			m.query = ""
			return m, nil

		case "enter":
			if m.onSubmit != nil {
				m.onSubmit(m.query)
			}
			m.active = false
			return m, nil

		case "ctrl+c":
			m.active = false
			m.query = ""
			return m, nil

		case "backspace":
			if m.query != "" {
				r := []rune(m.query)
				m.query = string(r[:len(r)-1])
			}

		default:
			if len(msg.String()) == 1 {
				m.query += msg.String()
			}
		}
	}

	return m, nil
}

func (m SearchModel) View() string {
	if !m.active {
		return ""
	}

	return styleActive.Render("  / " + m.query + "▌")
}

func (m *SearchModel) Activate() {
	m.active = true
	m.query = ""
}

func (m *SearchModel) Deactivate() {
	m.active = false
	m.query = ""
}

func (m SearchModel) IsActive() bool {
	return m.active
}

func (m SearchModel) GetQuery() string {
	return m.query
}

func (m *SearchModel) SetOnSubmit(fn func(string)) {
	m.onSubmit = fn
}

func (m *SearchModel) SetQuery(q string) {
	m.query = q
}

func (m SearchModel) RenderInline() string {
	if !m.active {
		return ""
	}
	return styleActive.Render("  / " + m.query + "▌")
}

func (m SearchModel) RenderOverlay() string {
	if !m.active {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(styleBorder.Render("  Search: " + m.query + "▌"))
	sb.WriteString("\n")
	sb.WriteString(styleFaint.Render("  Enter: search  Esc: cancel\n"))

	return sb.String()
}
