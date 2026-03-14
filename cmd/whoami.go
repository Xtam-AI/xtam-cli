package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/xtam-ai/xtam-cli/internal/auth"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the currently authenticated user",
	RunE: func(cmd *cobra.Command, args []string) error {
		stored, err := auth.LoadAuth()
		if err != nil {
			return err
		}

		fmt.Printf("Email:   %s\n", stored.Email)
		if stored.Name != "" {
			fmt.Printf("Name:    %s\n", stored.Name)
		}

		if time.Now().Before(stored.ExpiresAt) {
			fmt.Printf("Status:  authenticated (expires %s)\n", stored.ExpiresAt.Format(time.RFC3339))
		} else {
			fmt.Printf("Status:  token expired (will auto-refresh on next request)\n")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
