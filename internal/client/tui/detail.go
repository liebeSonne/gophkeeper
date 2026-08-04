package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
)

type detailFieldType int

const (
	fieldTypeText detailFieldType = iota
	fieldTypePassword
	fieldTypeCard
	fieldTypeFileID
)

type DetailField struct {
	Label  string
	Value  string
	Type   detailFieldType
	Copied bool
}

type DetailModel struct {
	ID        uuid.UUID
	Type      string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
	Fields    []DetailField
	selected  int
	width     int
	height    int
	message   string
	isFile    bool
}

func NewDetailModel() DetailModel {
	return DetailModel{}
}

func (m DetailModel) Init() tea.Cmd {
	return nil
}

func (m DetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case keyEsc:
			return m, nil
		case "j", keyDown:
			if m.selected < len(m.Fields)-1 {
				m.selected++
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
			}
		case "c":
			if m.selected >= 0 && m.selected < len(m.Fields) {
				field := &m.Fields[m.selected]
				if err := clipboard.WriteAll(field.Value); err == nil {
					field.Copied = true
					m.message = fmt.Sprintf("Copied %s to clipboard", field.Label)
				} else {
					m.message = "Failed to copy to clipboard"
				}
			}
		}
	}

	return m, nil
}

func (m DetailModel) View() string {
	var sb strings.Builder

	sb.WriteString(styleHeader.Render("  Details\n"))
	sb.WriteString("\n")

	fmt.Fprintf(&sb, "  %s: %s\n", styleBold.Render("ID"), m.ID.String())
	fmt.Fprintf(&sb, "  %s: %s\n", styleBold.Render("Type"), m.Type)
	if m.Title != "" {
		fmt.Fprintf(&sb, "  %s: %s\n", styleBold.Render("Title"), m.Title)
	}
	fmt.Fprintf(&sb, "  %s: %s\n", styleBold.Render("Created"), m.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&sb, "  %s: %s\n", styleBold.Render("Updated"), m.UpdatedAt.Format("2006-01-02 15:04:05"))
	sb.WriteString("\n")

	if len(m.Fields) > 0 {
		sb.WriteString(styleFaint.Render(strings.Repeat("-", 40)) + "\n")
		for i, field := range m.Fields {
			var value string
			switch field.Type {
			case fieldTypePassword:
				if i == m.selected {
					value = field.Value
				} else {
					value = strings.Repeat("•", len(field.Value))
				}
			default:
				value = field.Value
			}

			style := highlighted(i == m.selected)
			sb.WriteString(style.Render(fmt.Sprintf("  %s: %s\n",
				styleBold.Render(field.Label+":"),
				value)))

			if field.Copied {
				sb.WriteString(styleMuted.Render("    (copied)\n"))
			}
		}
	}

	sb.WriteString("\n")
	sb.WriteString(m.renderHelp())

	if m.message != "" {
		sb.WriteString("\n")
		sb.WriteString(styleActive.Render("  " + m.message + "\n"))
	}

	return sb.String()
}

func (m DetailModel) renderHelp() string {
	var keys []string

	if len(m.Fields) > 0 {
		keys = append(keys, styleKey.Render("j/k")+": field")
		keys = append(keys, styleKey.Render("c")+": copy")
	}
	keys = append(keys, styleKey.Render("esc")+": back")

	return styleFaint.Render("  " + strings.Join(keys, "  "))
}

func (m *DetailModel) SetMessage(msg string) {
	m.message = msg
}

func (m DetailModel) GetMessage() string {
	return m.message
}

func NewLoginPasswordDetail(id uuid.UUID, login, password string, created, updated time.Time) DetailModel {
	return DetailModel{
		ID:        id,
		Type:      "LOGIN_PASSWORD",
		CreatedAt: created,
		UpdatedAt: updated,
		Fields: []DetailField{
			{Label: "Login", Value: login, Type: fieldTypeText},
			{Label: "Password", Value: password, Type: fieldTypePassword},
		},
	}
}

func NewBankCardDetail(id uuid.UUID, cardNumber, cardHolder, cardExpiry, cardCvv string, created, updated time.Time) DetailModel {
	return DetailModel{
		ID:        id,
		Type:      "BANK_CARD",
		CreatedAt: created,
		UpdatedAt: updated,
		Fields: []DetailField{
			{Label: "Card", Value: cardNumber, Type: fieldTypeCard},
			{Label: "Holder", Value: cardHolder, Type: fieldTypeText},
			{Label: "Expiry", Value: cardExpiry, Type: fieldTypeText},
			{Label: "CVV", Value: cardCvv, Type: fieldTypePassword},
		},
	}
}

func NewTextDetail(id uuid.UUID, text string, created, updated time.Time) DetailModel {
	return DetailModel{
		ID:        id,
		Type:      "TEXT",
		CreatedAt: created,
		UpdatedAt: updated,
		Fields: []DetailField{
			{Label: "Text", Value: text, Type: fieldTypeText},
		},
	}
}

func NewFileDetail(id uuid.UUID, fileIDs []uuid.UUID, created, updated time.Time) DetailModel {
	var fields []DetailField
	for i, fid := range fileIDs {
		fields = append(fields, DetailField{
			Label: fmt.Sprintf("File %d", i+1),
			Value: fid.String(),
			Type:  fieldTypeFileID,
		})
	}
	if len(fields) == 0 {
		fields = append(fields, DetailField{Label: "Files", Value: "(none)", Type: fieldTypeText})
	}

	return DetailModel{
		ID:        id,
		Type:      "FILE",
		CreatedAt: created,
		UpdatedAt: updated,
		Fields:    fields,
	}
}

func NewFileItemDetail(id uuid.UUID, name, mimeType, status string, size int64, created, updated time.Time) DetailModel {
	var sizeStr string
	if size >= 0 {
		sizeStr = humanFileSize(uint64(size)) // nolint:gosec
	}

	return DetailModel{
		ID:        id,
		Type:      "FILE",
		Title:     name,
		CreatedAt: created,
		UpdatedAt: updated,
		isFile:    true,
		Fields: []DetailField{
			{Label: "Name", Value: name, Type: fieldTypeText},
			{Label: "Size", Value: sizeStr, Type: fieldTypeText},
			{Label: "Type", Value: mimeType, Type: fieldTypeText},
			{Label: "Status", Value: status, Type: fieldTypeText},
		},
	}
}
