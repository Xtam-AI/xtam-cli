package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/xtam-ai/xtam-cli/internal/auth"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with your @xtam.ai Google account",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if already logged in
		if existing, err := auth.LoadAuth(); err == nil {
			fmt.Printf("Already logged in as %s\n", existing.Email)
			fmt.Println("Run 'xtam logout' first to switch accounts.")
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		fmt.Println("Authenticating with Google...")

		tr, claims, err := auth.Login(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Login failed: %v\n", err)
			return err
		}

		// Save auth state
		stored := &auth.StoredAuth{
			IDToken:      tr.IDToken,
			RefreshToken: tr.RefreshToken,
			Email:        claims.Email,
			Name:         claims.Name,
			ExpiresAt:    time.Unix(claims.Exp, 0),
		}

		if err := auth.SaveAuth(stored); err != nil {
			return fmt.Errorf("failed to save credentials: %w", err)
		}

		fmt.Printf("\nAuthenticated as %s\n", claims.Email)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
