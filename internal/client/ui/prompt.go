package ui

import (
	"errors"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var ErrCancelled = errors.New("cancelled")

type promptModel struct {
	login    string
	password string
	field    int // 0=login, 1=password
	err      string
	quit     bool
}

func (m promptModel) Init() tea.Cmd {
	return nil
}

func (m promptModel) View() string {
	if m.quit {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("GophKeeper Authentication\n\n")

	if m.field == 0 {
		sb.WriteString("> Login: ")
		sb.WriteString(m.login)
		sb.WriteString(cursor())
	} else {
		sb.WriteString("  Login: ")
		sb.WriteString(m.login)
		sb.WriteString("\n")
	}

	if m.field == 1 {
		sb.WriteString("> Password: ")
		sb.WriteString(maskPassword(m.password))
		sb.WriteString(cursor())
	} else {
		sb.WriteString("  Password: ")
		sb.WriteString(maskPassword(m.password))
	}

	if m.err != "" {
		sb.WriteString("\n\n  ")
		sb.WriteString(m.err)
	}

	sb.WriteString("\n\n  Tab — next field  •  Enter — submit  •  Ctrl+C — cancel")
	return sb.String()
}

func (m promptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.Type {
	case tea.KeyCtrlC:
		m.quit = true
		return m, tea.Quit

	case tea.KeyTab:
		m.field = 1 - m.field
		m.err = ""
		return m, nil

	case tea.KeyEnter:
		m.err = ""

		if m.field == 0 {
			if m.login == "" {
				m.err = "login cannot be empty"
				return m, nil
			}
			m.field = 1
			return m, nil
		}

		if m.password == "" {
			m.err = "password cannot be empty"
			return m, nil
		}

		m.quit = true
		return m, tea.Quit

	case tea.KeyBackspace:
		if m.field == 0 {
			if m.login != "" {
				m.login = m.login[:len(m.login)-1]
			}
		} else {
			if m.password != "" {
				m.password = m.password[:len(m.password)-1]
			}
		}
		m.err = ""
		return m, nil

	default:
		if keyMsg.Type == tea.KeyRunes {
			input := string(keyMsg.Runes)
			if m.field == 0 {
				m.login += input
			} else {
				m.password += input
			}
			m.err = ""
		}
		return m, nil
	}
}

func cursor() string {
	return "█"
}

func maskPassword(p string) string {
	masked := strings.Repeat("•", len(p))
	return masked
}

func PromptCredentials() (login, password string, err error) {
	initialModel := promptModel{}
	p := tea.NewProgram(initialModel, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return "", "", fmt.Errorf("prompt failed: %w", err)
	}

	m := result.(promptModel)
	if m.quit && m.err != "" {
		return "", "", errors.New(m.err)
	}

	if m.login == "" || m.password == "" {
		return "", "", ErrCancelled
	}

	return m.login, m.password, nil
}
