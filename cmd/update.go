package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xtam-ai/xtam-cli/internal/installer"
	"github.com/xtam-ai/xtam-cli/internal/registry"
	"github.com/xtam-ai/xtam-cli/internal/state"
)

var updateAll bool

var updateCmd = &cobra.Command{
	Use:   "update [artifact]",
	Short: "Update an installed artifact (or all with --all)",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := state.Load()
		if err != nil {
			return err
		}

		client := registry.NewClient()

		var names []string
		if updateAll {
			for _, a := range st.List() {
				names = append(names, a.Name)
			}
			if len(names) == 0 {
				fmt.Println("No artifacts installed.")
				return nil
			}
		} else if len(args) > 0 {
			names = []string{args[0]}
		} else {
			return fmt.Errorf("specify an artifact name or use --all")
		}

		updated := 0
		for _, name := range names {
			existing := st.Get(name)
			if existing == nil {
				fmt.Printf("Skipping %s: not installed\n", name)
				continue
			}

			m, err := client.FetchManifest(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Skipping %s: %v\n", name, err)
				continue
			}

			if existing.Version == m.Version && existing.SHA256 == m.SHA256 {
				fmt.Printf("%s is up to date (v%s)\n", name, m.Version)
				continue
			}

			fmt.Printf("Updating %s: v%s → v%s\n", name, existing.Version, m.Version)

			archivePath, err := client.DownloadAndVerify(name, m.SHA256)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to download %s: %v\n", name, err)
				continue
			}

			inst, err := installer.ForType(m.Type)
			if err != nil {
				os.Remove(archivePath)
				fmt.Fprintf(os.Stderr, "Failed for %s: %v\n", name, err)
				continue
			}

			installPath, err := inst.Install(m, archivePath)
			os.Remove(archivePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to install %s: %v\n", name, err)
				continue
			}

			st.Record(m, installPath)
			updated++
			fmt.Printf("Updated %s → v%s\n", name, m.Version)
		}

		if err := st.Save(); err != nil {
			return fmt.Errorf("failed to save state: %w", err)
		}

		if updated > 0 {
			fmt.Printf("\n%d artifact(s) updated.\n", updated)
		}

		return nil
	},
}

func init() {
	updateCmd.Flags().BoolVar(&updateAll, "all", false, "Update all installed artifacts")
	rootCmd.AddCommand(updateCmd)
}
