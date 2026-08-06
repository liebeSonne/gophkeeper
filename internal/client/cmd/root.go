package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/liebeSonne/gophkeeper/internal/client/app"
	clientconfig "github.com/liebeSonne/gophkeeper/internal/client/config"
)

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

var rootCmd = &cobra.Command{
	Use:   "gk",
	Short: "GophKeeper - password manager CLI client",
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		if cmd.Name() == "version" {
			return nil
		}
		logLevel, _ := cmd.InheritedFlags().GetString("log-level")
		_, err := app.EnsureInitialized(logLevel)
		return err
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func SetVersion(version, date, commit string) {
	buildVersion = version
	buildDate = date
	buildCommit = commit
}

func Setup() {
	initCmd.Flags().StringVar(&initServerAddress, "server-address", "", "server address (e.g. http://localhost:8080)")
	initCmd.Flags().StringVar(&initStoragePath, "storage-path", "", "path to SQLite storage file")

	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(registerCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(refreshCmd)

	initDataFlags()
	dataCmd.AddCommand(dataListCmd)
	dataCmd.AddCommand(dataGetCmd)
	dataCmd.AddCommand(dataCreateCmd)
	dataCmd.AddCommand(dataUpdateCmd)
	dataCmd.AddCommand(dataDeleteCmd)

	initFileFlags()
	fileCmd.AddCommand(fileUploadCmd)
	fileCmd.AddCommand(fileDownloadCmd)
	fileCmd.AddCommand(fileListCmd)
	fileCmd.AddCommand(fileDeleteCmd)

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(dataCmd)
	rootCmd.AddCommand(fileCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(tuiCmd)
	rootCmd.PersistentFlags().String("log-level", clientconfig.DefaultLogLevel, "log level (debug, info, warn, error)")
}
