package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type loginState int

const (
	loginFieldServer loginState = iota
	loginFieldLogin
	loginFieldPassword
	loginStateConnecting
	loginStateError
)

type LoginModel struct {
	server   string
	login    string
	password string
	state    loginState
	cursor   int
	err      error
	width    int
	height   int
}

func NewLoginModel() LoginModel {
	return LoginModel{
		server: "http://localhost:8080",
		state:  loginFieldServer,
		cursor: 0,
	}
}

func (m LoginModel) Init() tea.Cmd {
	return nil
}

func (m LoginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			if m.state == loginFieldPassword {
				if m.login == "" || m.password == "" {
					m.err = fmt.Errorf("login and password are required")
					m.state = loginStateError
					return m, nil
				}
				m.state = loginStateConnecting
				return m, nil
			}
			m.nextField()
			return m, nil

		case tea.KeyTab:
			m.nextField()
			return m, nil

		case tea.KeyShiftTab:
			m.prevField()
			return m, nil

		case tea.KeyEscape:
			return m, tea.Quit

		case tea.KeyBackspace:
			m.backspace()

		default:
			if len(msg.String()) == 1 {
				m.appendRune(msg.String())
			}
		}
	}

	return m, nil
}

func (m *LoginModel) nextField() {
	switch m.state {
	case loginFieldServer:
		m.state = loginFieldLogin
	case loginFieldLogin:
		m.state = loginFieldPassword
	case loginFieldPassword:
		m.state = loginFieldServer
	}
}

func (m *LoginModel) prevField() {
	switch m.state {
	case loginFieldServer:
		m.state = loginFieldPassword
	case loginFieldLogin:
		m.state = loginFieldServer
	case loginFieldPassword:
		m.state = loginFieldLogin
	}
}

func (m *LoginModel) getCurrentField() *string {
	switch m.state {
	case loginFieldServer:
		return &m.server
	case loginFieldLogin:
		return &m.login
	case loginFieldPassword:
		return &m.password
	default:
		return &m.login
	}
}

func (m *LoginModel) appendRune(r string) {
	field := m.getCurrentField()
	if field == nil {
		return
	}
	*field += r
}

func (m *LoginModel) backspace() {
	field := m.getCurrentField()
	if field == nil {
		return
	}
	s := *field
	if s == "" {
		return
	}
	runes := []rune(s)
	*field = string(runes[:len(runes)-1])
}

func (m LoginModel) View() string {
	if m.state == loginStateConnecting {
		return m.renderConnecting()
	}

	if m.state == loginStateError && m.err != nil {
		return m.renderError()
	}

	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(styleHeader.Render("    GophKeeper Login\n"))
	sb.WriteString("\n")

	sb.WriteString(m.renderField("Server", m.server, m.state == loginFieldServer))
	sb.WriteString("\n\n")

	sb.WriteString(m.renderField("Login", m.login, m.state == loginFieldLogin))
	sb.WriteString("\n\n")

	sb.WriteString(m.renderField("Password", m.maskPassword(m.password), m.state == loginFieldPassword))
	sb.WriteString("\n\n")

	sb.WriteString(styleFaint.Render("  Tab: next field  Enter: connect  Esc: quit\n"))

	return sb.String()
}

func (m LoginModel) renderField(label, value string, active bool) string {
	cursor := ""
	if active {
		cursor = "▌"
	}
	prefix := styleBold.Render("  " + label + ": ")
	if active {
		prefix = styleActive.Render("  " + label + ": ")
	}
	return prefix + styleInput.Render(value+cursor)
}

func (m LoginModel) maskPassword(p string) string {
	if p == "" {
		return ""
	}
	return strings.Repeat("•", len([]rune(p)))
}

func (m LoginModel) renderConnecting() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(styleHeader.Render("    Connecting...\n"))
	sb.WriteString("\n")
	sb.WriteString(styleFaint.Render("  Connecting to " + m.server + "\n"))
	return sb.String()
}

func (m LoginModel) renderError() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(styleError.Render("    Error: " + m.err.Error() + "\n"))
	sb.WriteString("\n")
	sb.WriteString(styleFaint.Render("  Press Enter to try again\n"))
	return sb.String()
}

func (m LoginModel) GetServer() string {
	return m.server
}

func (m LoginModel) GetLogin() string {
	return m.login
}

func (m LoginModel) GetPassword() string {
	return m.password
}

func (m LoginModel) IsConnecting() bool {
	return m.state == loginStateConnecting
}

func (m LoginModel) HasError() bool {
	return m.state == loginStateError
}

func (m *LoginModel) SetError(err error) {
	m.err = err
	m.state = loginStateError
}

func (m *LoginModel) Reset() {
	m.state = loginFieldServer
	m.err = nil
}
