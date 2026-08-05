package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type editorState int

const (
	editorTypeSelect editorState = iota
	editorFieldInput
	editorDone
)

type EditorModel struct {
	dataType   string
	fields     map[string]string
	fieldOrder []string
	selected   int
	state      editorState
	width      int
	height     int
	err        error
	submit     func() error
}

var dataTypes = []string{dataTypeLoginPassword, dataTypeBankCard, dataTypeText, dataTypeFile}

var fieldTemplates = map[string][]string{
	dataTypeLoginPassword: {fieldNameLogin, fieldNamePassword, fieldNameMetadata},
	dataTypeBankCard:      {fieldNameCardNumber, fieldNameCardHolder, fieldNameCardExpiry, fieldNameCardCVV, fieldNameMetadata},
	dataTypeText:          {fieldNameText, fieldNameMetadata},
	dataTypeFile:          {"File IDs (comma-separated)", fieldNameMetadata},
}

func NewEditorModel() EditorModel {
	return EditorModel{
		fields: make(map[string]string),
		state:  editorTypeSelect,
	}
}

func (m EditorModel) Init() tea.Cmd {
	return nil
}

func (m EditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.state == editorTypeSelect {
				return m, nil
			}
			m.state = editorTypeSelect
			return m, nil

		case "j", "down":
			m.selected++
			if m.selected >= len(m.getOptions()) {
				m.selected = len(m.getOptions()) - 1
			}
		case "k", "up":
			m.selected--
			if m.selected < 0 {
				m.selected = 0
			}
		case "enter":
			return m.handleEnter()

		case "backspace":
			m.handleBackspace()

		default:
			if m.state == editorFieldInput && len(msg.String()) == 1 {
				m.handleInput(msg.String())
			}
		}
	}

	return m, nil
}

func (m *EditorModel) getOptions() []string {
	if m.state == editorTypeSelect {
		return dataTypes
	}
	return m.fieldOrder
}

func (m *EditorModel) handleEnter() (tea.Model, tea.Cmd) {
	if m.state == editorTypeSelect {
		if m.selected < len(dataTypes) {
			m.dataType = dataTypes[m.selected]
			m.fieldOrder = fieldTemplates[m.dataType]
			for _, field := range m.fieldOrder {
				m.fields[field] = ""
			}
			m.state = editorFieldInput
			m.selected = 0
		}
		return *m, nil
	}

	if m.selected < len(m.fieldOrder)-1 {
		return *m, nil
	}

	if err := m.validate(); err != nil {
		m.err = err
		return *m, nil
	}

	if m.submit != nil {
		if err := m.submit(); err != nil {
			m.err = err
			return *m, nil
		}
		m.state = editorDone
		return *m, nil
	}

	m.state = editorDone
	return *m, nil
}

func (m *EditorModel) handleBackspace() {
	if m.state != editorFieldInput {
		return
	}
	if m.selected >= len(m.fieldOrder) {
		return
	}
	field := m.fieldOrder[m.selected]
	val := m.fields[field]
	if val == "" {
		return
	}
	r := []rune(val)
	m.fields[field] = string(r[:len(r)-1])
}

func (m *EditorModel) handleInput(s string) {
	if m.selected >= len(m.fieldOrder) {
		return
	}
	field := m.fieldOrder[m.selected]
	m.fields[field] += s
}

func (m *EditorModel) validate() error {
	if m.dataType == "" {
		return fmt.Errorf("data type is required")
	}

	switch m.dataType {
	case dataTypeLoginPassword:
		if m.fields[fieldNameLogin] == "" {
			return fmt.Errorf("login is required")
		}
		if m.fields[fieldNamePassword] == "" {
			return fmt.Errorf("password is required")
		}
	case dataTypeBankCard:
		if m.fields[fieldNameCardNumber] == "" {
			return fmt.Errorf("card number is required")
		}
	case dataTypeText:
		if m.fields[fieldNameText] == "" {
			return fmt.Errorf("text is required")
		}
	}

	return nil
}

func (m EditorModel) View() string {
	var sb strings.Builder

	if m.state == editorDone {
		sb.WriteString(styleHeader.Render("  Saved!\n"))
		sb.WriteString(styleFaint.Render("  Press Esc to go back\n"))
		return sb.String()
	}

	if m.state == editorTypeSelect {
		return m.renderTypeSelect()
	}

	return m.renderFields()
}

func (m EditorModel) renderTypeSelect() string {
	var sb strings.Builder
	sb.WriteString(styleHeader.Render("  Select Data Type\n"))
	sb.WriteString("\n")

	for i, dt := range dataTypes {
		style := highlighted(i == m.selected)
		label := m.getDataTypeLabel(dt)
		sb.WriteString(style.Render(fmt.Sprintf("  %s %s\n",
			getBullet(i == m.selected),
			label)))
	}

	sb.WriteString("\n")
	sb.WriteString(styleFaint.Render("  j/k: select  Enter: choose  Esc: cancel\n"))

	return sb.String()
}

func (m EditorModel) renderFields() string {
	var sb strings.Builder

	sb.WriteString(styleHeader.Render(fmt.Sprintf("  Edit: %s\n", m.dataType)))
	sb.WriteString("\n")

	for i, field := range m.fieldOrder {
		style := highlighted(i == m.selected)
		value := m.fields[field]

		if field == fieldNamePassword || field == "Card CVV" {
			if value != "" {
				value = strings.Repeat("•", len([]rune(value)))
			}
		}

		cursor := ""
		if i == m.selected {
			cursor = "▌"
		}

		sb.WriteString(style.Render(fmt.Sprintf("  %s %s: %s%s\n",
			getBullet(i == m.selected),
			field,
			value,
			cursor)))
	}

	if m.err != nil {
		sb.WriteString("\n")
		sb.WriteString(styleError.Render("  " + m.err.Error() + "\n"))
	}

	sb.WriteString("\n")
	sb.WriteString(styleFaint.Render("  j/k: navigate  Enter: next/save  Esc: back\n"))

	return sb.String()
}

func (m EditorModel) getDataTypeLabel(dt string) string {
	labels := map[string]string{
		dataTypeLoginPassword: "Login & Password",
		dataTypeBankCard:      "Bank Card",
		dataTypeText:          "Text Note",
		dataTypeFile:          "File Reference",
	}
	if label, ok := labels[dt]; ok {
		return label
	}
	return dt
}

func getBullet(active bool) string {
	if active {
		return "▸"
	}
	return " "
}

func (m EditorModel) GetDataType() string {
	return m.dataType
}

func (m EditorModel) GetFields() map[string]string {
	return m.fields
}

func (m EditorModel) IsDone() bool {
	return m.state == editorDone
}

func (m *EditorModel) Reset() {
	m.dataType = ""
	m.fields = make(map[string]string)
	m.fieldOrder = nil
	m.selected = 0
	m.state = editorTypeSelect
	m.err = nil
}

func (m *EditorModel) SetSubmit(fn func() error) {
	m.submit = fn
}
