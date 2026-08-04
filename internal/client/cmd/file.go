package cmd

import (
	"github.com/spf13/cobra"
)

var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "File management commands",
}

var fileUploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a file",
	RunE:  func(_ *cobra.Command, _ []string) error { return nil },
}

var fileDownloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Download a file",
	RunE:  func(_ *cobra.Command, _ []string) error { return nil },
}

var fileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List files",
	RunE:  func(_ *cobra.Command, _ []string) error { return nil },
}

var fileDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a file",
	RunE:  func(_ *cobra.Command, _ []string) error { return nil },
}
