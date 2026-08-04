package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	styleBold      = lipgloss.NewStyle().Bold(true)
	styleFaint     = lipgloss.NewStyle().Faint(true)
	styleSelected  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11"))
	styleActive    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	styleMuted     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleError     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	styleHeader    = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	styleBorder    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	styleStatusBar = lipgloss.NewStyle().Bold(true).Padding(0, 1).Height(1)
	styleInput     = lipgloss.NewStyle().Padding(0, 1)
	styleKey       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
)

func separator() string {
	return styleFaint.Render(" | ")
}

func highlighted(selected bool) lipgloss.Style {
	if selected {
		return styleSelected
	}
	return lipgloss.NewStyle()
}

func tabStyle(active bool) lipgloss.Style {
	if active {
		return styleActive
	}
	return styleFaint
}
