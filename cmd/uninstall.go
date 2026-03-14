package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xtam-ai/xtam-cli/internal/installer"
	"github.com/xtam-ai/xtam-cli/internal/manifest"
	"github.com/xtam-ai/xtam-cli/internal/state"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <artifact>",
	Short: "Remove an installed artifact",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		st, err := state.Load()
		if err != nil {
			return err
		}

		existing := st.Get(name)
		if existing == nil {
			return fmt.Errorf("'%s' is not installed", name)
		}

		// Build a minimal manifest for the uninstaller
		m := &manifest.Manifest{
			Name: existing.Name,
			Type: existing.Type,
			Install: manifest.Install{
				Target:    existing.Type,
				SkillName: existing.Name, // works for skills
			},
		}

		inst, err := installer.ForType(existing.Type)
		if err != nil {
			return err
		}

		if err := inst.Uninstall(m); err != nil {
			return fmt.Errorf("uninstall failed: %w", err)
		}

		st.Remove(name)
		if err := st.Save(); err != nil {
			return fmt.Errorf("failed to update state: %w", err)
		}

		fmt.Printf("Uninstalled %s\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
