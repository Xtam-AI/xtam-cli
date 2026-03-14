package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xtam-ai/xtam-cli/internal/auth"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := auth.DeleteAuth(); err != nil {
			return err
		}
		fmt.Println("Logged out.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
