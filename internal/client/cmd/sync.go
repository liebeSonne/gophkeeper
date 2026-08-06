package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/liebeSonne/gophkeeper/internal/client/app"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize local data with the server",
	Long:  "Explicitly synchronize local data with the server. Pushes pending changes and pulls server updates.",
	RunE: func(_ *cobra.Command, _ []string) error {
		a := app.Get()
		if a == nil {
			return app.ErrNotInitialized
		}
		if a.Sync == nil {
			fmt.Println("Sync is not configured. Ensure server address is set.")
			return nil
		}
		fmt.Println("Syncing with server...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := a.Sync.Sync(ctx)
		if err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		fmt.Println("Sync completed successfully.")
		return nil
	},
}
