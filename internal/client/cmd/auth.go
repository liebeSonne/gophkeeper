package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	apiClient "github.com/liebeSonne/gophkeeper/internal/client/adapter/gophkeeper"
	"github.com/liebeSonne/gophkeeper/internal/client/app"
	"github.com/liebeSonne/gophkeeper/internal/client/model"
	clientui "github.com/liebeSonne/gophkeeper/internal/client/ui"
	gophkeeper "github.com/liebeSonne/gophkeeper/pkg/client/gophkeeper"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to the server",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true

		a, err := app.EnsureInitialized()
		if err != nil {
			return err
		}

		login, password, err := getCredentials(cmd)
		if err != nil {
			return err
		}

		client, err := newAPIClient(a)
		if err != nil {
			return err
		}

		resp, err := client.LoginUser(cmd.Context(), login, password)
		if err != nil {
			return err
		}

		token := convertTokenResponse(resp.JSON200)

		if err := a.Storage.SaveTokens(token); err != nil {
			return fmt.Errorf("save tokens: %w", err)
		}

		client.SetAuthToken(token.AccessToken)

		fmt.Println("Login successful.")
		return nil
	},
}

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new user",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true

		a, err := app.EnsureInitialized()
		if err != nil {
			return err
		}

		login, password, err := getCredentials(cmd)
		if err != nil {
			return err
		}

		client, err := newAPIClient(a)
		if err != nil {
			return err
		}

		_, err = client.RegisterUser(cmd.Context(), login, password)
		if err != nil {
			return err
		}

		fmt.Println("User registered successfully.")
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logout and clear stored tokens",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true

		a, err := app.EnsureInitialized()
		if err != nil {
			return err
		}

		if err := a.Storage.ClearTokens(); err != nil {
			return fmt.Errorf("clear tokens: %w", err)
		}

		fmt.Println("Logged out successfully.")
		return nil
	},
}

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh access token",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true

		a, err := app.EnsureInitialized()
		if err != nil {
			return err
		}

		tokens, err := a.Storage.GetTokens()
		if err != nil {
			return fmt.Errorf("get tokens: %w", err)
		}
		if tokens == nil {
			return fmt.Errorf("no tokens stored, please login first")
		}

		client, err := newAPIClient(a)
		if err != nil {
			return err
		}

		resp, err := client.RefreshTokenAPI(cmd.Context(), tokens.RefreshToken)
		if err != nil {
			return err
		}

		newToken := convertTokenResponse(resp.JSON200)

		if err := a.Storage.SaveTokens(newToken); err != nil {
			return fmt.Errorf("save tokens: %w", err)
		}

		client.SetAuthToken(newToken.AccessToken)

		fmt.Println("Token refreshed successfully.")
		return nil
	},
}

func getCredentials(cmd *cobra.Command) (login, password string, err error) {
	login, _ = cmd.Flags().GetString("login")
	password, _ = cmd.Flags().GetString("password")

	if login != "" && password != "" {
		return login, password, nil
	}

	if login == "" && password == "" {
		return clientui.PromptCredentials()
	}

	return "", "", fmt.Errorf("both --login and --password must be provided together")
}

func convertTokenResponse(resp *gophkeeper.TokenResponse) model.Token {
	return model.Token{
		AccessToken:          resp.AccessToken,
		RefreshToken:         resp.RefreshToken,
		AccessTokenExpiresAt: time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second),
		RefreshExpiresAt:     time.Now().Add(time.Duration(resp.RefreshTokenExpiresIn) * time.Second),
	}
}

func newAPIClient(a *app.App) (*apiClient.Client, error) {
	client, err := apiClient.NewClient(a.Config.ServerAddress)
	if err != nil {
		return nil, fmt.Errorf("create API client: %w", err)
	}

	tokens, err := a.Storage.GetTokens()
	if err != nil {
		return nil, fmt.Errorf("get tokens: %w", err)
	}
	if tokens != nil {
		client.SetAuthToken(tokens.AccessToken)
	}

	return client, nil
}
