package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/liebeSonne/gophkeeper/internal/client/tui"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Start TUI interface",
	RunE:  runTUI,
}

func runTUI(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	app, err := tui.NewApp()
	if err != nil {
		return fmt.Errorf("initialize TUI: %w", err)
	}

	if _, err := tea.NewProgram(app, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		return err
	}

	return nil
}
