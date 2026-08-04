package cmd

import (
	"github.com/spf13/cobra"
)

var dataCmd = &cobra.Command{
	Use:   "data",
	Short: "Data management commands",
}

var dataListCmd = &cobra.Command{
	Use:   "list",
	Short: "List data entries",
	RunE:  func(_ *cobra.Command, _ []string) error { return nil },
}

var dataGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a data entry by ID",
	RunE:  func(_ *cobra.Command, _ []string) error { return nil },
}

var dataCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new data entry",
	RunE:  func(_ *cobra.Command, _ []string) error { return nil },
}

var dataUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update a data entry",
	RunE:  func(_ *cobra.Command, _ []string) error { return nil },
}

var dataDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a data entry",
	RunE:  func(_ *cobra.Command, _ []string) error { return nil },
}
