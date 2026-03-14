package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xtam-ai/xtam-cli/internal/registry"
)

var infoCmd = &cobra.Command{
	Use:   "info <artifact>",
	Short: "Show details about an artifact",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		client := registry.NewClient()
		m, err := client.FetchManifest(name)
		if err != nil {
			return err
		}

		fmt.Printf("Name:        %s\n", m.Name)
		fmt.Printf("Version:     %s\n", m.Version)
		fmt.Printf("Type:        %s\n", m.Type)
		fmt.Printf("Description: %s\n", m.Description)
		fmt.Printf("Author:      %s <%s>\n", m.Author.Name, m.Author.Email)
		fmt.Printf("Archive:     %s\n", m.Archive)
		fmt.Printf("SHA256:      %s\n", m.SHA256)
		if m.SizeBytes > 0 {
			fmt.Printf("Size:        %s\n", formatBytes(m.SizeBytes))
		}
		if m.Changelog != "" {
			fmt.Printf("\nChangelog:\n%s\n", m.Changelog)
		}

		return nil
	},
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
