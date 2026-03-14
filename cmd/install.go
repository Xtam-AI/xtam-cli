package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xtam-ai/xtam-cli/internal/installer"
	"github.com/xtam-ai/xtam-cli/internal/registry"
	"github.com/xtam-ai/xtam-cli/internal/state"
)

var installCmd = &cobra.Command{
	Use:   "install <artifact>[@version]",
	Short: "Install an artifact from the registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		// TODO: handle @version suffix
		if idx := strings.LastIndex(name, "@"); idx > 0 {
			name = name[:idx]
			// version = name[idx+1:]
		}

		client := registry.NewClient()

		// Fetch manifest
		fmt.Printf("Fetching %s...\n", name)
		m, err := client.FetchManifest(name)
		if err != nil {
			return fmt.Errorf("artifact '%s' not found: %w", name, err)
		}

		// Check if already installed at same version
		st, err := state.Load()
		if err != nil {
			return err
		}
		if existing := st.Get(name); existing != nil && existing.Version == m.Version {
			fmt.Printf("Already installed: %s v%s\n", name, m.Version)
			return nil
		}

		// Download and verify
		fmt.Printf("Downloading %s v%s (%s)...\n", m.Name, m.Version, m.Type)
		archivePath, err := client.DownloadAndVerify(name, m.SHA256)
		if err != nil {
			return err
		}
		defer os.Remove(archivePath)

		// Install
		inst, err := installer.ForType(m.Type)
		if err != nil {
			return err
		}

		installPath, err := inst.Install(m, archivePath)
		if err != nil {
			return fmt.Errorf("installation failed: %w", err)
		}

		// Record in state
		st.Record(m, installPath)
		if err := st.Save(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not save install state: %v\n", err)
		}

		fmt.Printf("Installed %s v%s → %s\n", m.Name, m.Version, installPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
