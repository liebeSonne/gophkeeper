package cmd

import (
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to the server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return cmd.Flags().Parse(args)
	},
}

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new user",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return cmd.Flags().Parse(args)
	},
}
