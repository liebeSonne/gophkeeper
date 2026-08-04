package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	RunE: func(_ *cobra.Command, _ []string) error {
		fmt.Printf("Version: %s\n", buildVersion)
		fmt.Printf("Build date: %s\n", buildDate)
		fmt.Printf("Build commit: %s\n", buildCommit)
		return nil
	},
}
