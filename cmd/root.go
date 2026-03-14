package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "xtam",
	Short: "XTAM private artifact registry CLI",
	Long:  "Install and manage private skills, tools, MCP servers, configs, and templates from the XTAM registry.",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
