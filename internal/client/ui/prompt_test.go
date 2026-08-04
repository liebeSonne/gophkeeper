package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptModel_Success(t *testing.T) {
	m := promptModel{}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("user")})
	m = result.(promptModel)
	assert.Equal(t, "user", m.login)
	assert.Equal(t, 0, m.field)

	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(promptModel)
	assert.Equal(t, 1, m.field)
	assert.Empty(t, m.err)

	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("pass")})
	m = result.(promptModel)
	assert.Equal(t, "pass", m.password)

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(promptModel)
	assert.True(t, m.quit)
	assert.NotNil(t, cmd)
}

func TestPromptModel_EmptyLogin(t *testing.T) {
	m := promptModel{}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(promptModel)
	assert.Equal(t, 0, m.field)
	assert.Equal(t, "login cannot be empty", m.err)
}

func TestPromptModel_EmptyPassword(t *testing.T) {
	m := promptModel{}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("user")})
	m = result.(promptModel)
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(promptModel)
	assert.Equal(t, 1, m.field)

	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(promptModel)
	assert.Equal(t, 1, m.field)
	assert.Equal(t, "password cannot be empty", m.err)
}

func TestPromptModel_Cancel(t *testing.T) {
	m := promptModel{}

	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = result.(promptModel)
	assert.True(t, m.quit)
	assert.NotNil(t, cmd)
}

func TestPromptModel_Backspace(t *testing.T) {
	m := promptModel{}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("abc")})
	m = result.(promptModel)
	assert.Equal(t, "abc", m.login)

	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = result.(promptModel)
	assert.Equal(t, "ab", m.login)
}

func TestPromptModel_Tab(t *testing.T) {
	m := promptModel{}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("user")})
	m = result.(promptModel)
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = result.(promptModel)
	assert.Equal(t, 1, m.field)

	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = result.(promptModel)
	assert.Equal(t, 0, m.field)
}

func TestMaskPassword(t *testing.T) {
	assert.Equal(t, "••••", maskPassword("pass"))
	assert.Equal(t, "", maskPassword(""))
	assert.Equal(t, "••••••••", maskPassword("12345678"))
}

func TestView(t *testing.T) {
	m := promptModel{
		login: "user",
		field: 0,
	}

	view := m.View()
	assert.Contains(t, view, "GophKeeper Authentication")
	assert.Contains(t, view, "> Login: user")
	assert.Contains(t, view, "█")
}

func TestView_PasswordField(t *testing.T) {
	m := promptModel{
		login:    "user",
		password: "pass",
		field:    1,
	}

	view := m.View()
	assert.Contains(t, view, "  Login: user")
	assert.Contains(t, view, "> Password: ••••")
	assert.NotContains(t, view, "pass")
}

func TestView_Error(t *testing.T) {
	m := promptModel{
		err: "login cannot be empty",
	}

	view := m.View()
	assert.Contains(t, view, "login cannot be empty")
}

func TestPromptModel_LoginNotEmptyValidation(t *testing.T) {
	m := promptModel{}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(promptModel)
	require.Equal(t, 0, m.field)
	require.Equal(t, "login cannot be empty", m.err)

	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = result.(promptModel)
	require.Empty(t, m.err)
	require.Equal(t, "a", m.login)
}

func TestPromptModel_PasswordNotEmptyValidation(t *testing.T) {
	m := promptModel{login: "user", field: 1}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(promptModel)
	require.Equal(t, 1, m.field)
	require.Equal(t, "password cannot be empty", m.err)

	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = result.(promptModel)
	require.Empty(t, m.err)
	require.Equal(t, "p", m.password)
}

func TestPromptModel_ViewQuit(t *testing.T) {
	m := promptModel{quit: true}

	view := m.View()
	assert.Empty(t, view)
}

func TestPromptModel_TabClearsError(t *testing.T) {
	m := promptModel{err: "some error"}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = result.(promptModel)
	assert.Empty(t, m.err)
	assert.Equal(t, 1, m.field)
}

func TestPromptModel_EnterClearsError(t *testing.T) {
	m := promptModel{login: "user", err: "some error"}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = result.(promptModel)
	assert.Empty(t, m.err)
	assert.Equal(t, 1, m.field)
}

func TestPromptModel_InputClearsError(t *testing.T) {
	m := promptModel{err: "some error"}

	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = result.(promptModel)
	assert.Empty(t, m.err)
	assert.Equal(t, "a", m.login)
}
