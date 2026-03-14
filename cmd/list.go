package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/xtam-ai/xtam-cli/internal/state"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "installed"},
	Short:   "List installed artifacts",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := state.Load()
		if err != nil {
			return err
		}

		artifacts := st.List()
		if len(artifacts) == 0 {
			fmt.Println("No artifacts installed.")
			fmt.Println("Run 'xtam catalog' to see available artifacts.")
			return nil
		}

		sort.Slice(artifacts, func(i, j int) bool {
			return artifacts[i].Name < artifacts[j].Name
		})

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tVERSION\tINSTALLED\tPATH")
		fmt.Fprintln(w, "----\t----\t-------\t---------\t----")

		for _, a := range artifacts {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				a.Name,
				a.Type,
				a.Version,
				a.InstalledAt.Format("2006-01-02"),
				a.InstallPath,
			)
		}

		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
