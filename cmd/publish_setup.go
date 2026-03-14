package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xtam-ai/xtam-cli/internal/publisher"
)

var publishSetupCmd = &cobra.Command{
	Use:   "publish-setup",
	Short: "Configure a GitHub token for publishing artifacts",
	Long: `Store a GitHub PAT that has write access to the Xtam-AI/registry repo.

Create a fine-grained PAT at:
  GitHub → Settings → Developer settings → Fine-grained personal access tokens

Required permissions:
  - Repository: Xtam-AI/registry
  - Contents: Read and write`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print("GitHub PAT (with write access to Xtam-AI/registry): ")
		reader := bufio.NewReader(os.Stdin)
		token, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		token = strings.TrimSpace(token)

		if token == "" {
			return fmt.Errorf("no token provided")
		}

		if err := publisher.SavePublishToken(token); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}

		fmt.Println("Publish token saved to ~/.xtam/publish.json")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(publishSetupCmd)
}
