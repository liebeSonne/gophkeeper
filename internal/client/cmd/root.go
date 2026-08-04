package cmd

import (
	"os"

	"github.com/spf13/cobra"

	clientconfig "github.com/liebeSonne/gophkeeper/internal/client/config"
)

var buildVersion = "N/A"
var buildDate = "N/A"
var buildCommit = "N/A"

var rootCmd = &cobra.Command{
	Use:   "gk",
	Short: "GophKeeper - password manager CLI client",
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
	// init
	initCmd.Flags().StringVar(&initServerAddress, "server-address", "", "server address (e.g. http://localhost:8080)")
	initCmd.Flags().StringVar(&initStoragePath, "storage-path", "", "path to SQLite storage file")

	// register
	registerCmd.Flags().String("login", "", "username")
	registerCmd.Flags().String("password", "", "password")

	// login
	loginCmd.Flags().String("login", "", "username")
	loginCmd.Flags().String("password", "", "password")

	// auth
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(registerCmd)

	// data
	dataCmd.AddCommand(dataListCmd)
	dataCmd.AddCommand(dataGetCmd)
	dataCmd.AddCommand(dataCreateCmd)
	dataCmd.AddCommand(dataUpdateCmd)
	dataCmd.AddCommand(dataDeleteCmd)

	// file
	fileCmd.AddCommand(fileUploadCmd)
	fileCmd.AddCommand(fileDownloadCmd)
	fileCmd.AddCommand(fileListCmd)
	fileCmd.AddCommand(fileDeleteCmd)

	// root
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(dataCmd)
	rootCmd.AddCommand(fileCmd)
	rootCmd.PersistentFlags().String("log-level", clientconfig.DefaultLogLevel, "log level (debug, info, warn, error)")
}
