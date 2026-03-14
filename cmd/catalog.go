package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/xtam-ai/xtam-cli/internal/registry"
)

var (
	catalogTypeFilter string
	catalogTagFilter  string
)

var catalogCmd = &cobra.Command{
	Use:     "catalog",
	Aliases: []string{"search"},
	Short:   "List available artifacts in the registry",
	RunE: func(cmd *cobra.Command, args []string) error {
		client := registry.NewClient()
		catalog, err := client.FetchCatalog()
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tVERSION\tDESCRIPTION")
		fmt.Fprintln(w, "----\t----\t-------\t-----------")

		for _, a := range catalog.Artifacts {
			// Filter by type
			if catalogTypeFilter != "" && string(a.Type) != catalogTypeFilter {
				continue
			}
			// Filter by tag
			if catalogTagFilter != "" && !containsTag(a.Tags, catalogTagFilter) {
				continue
			}
			// Filter by search term (positional arg)
			if len(args) > 0 {
				query := strings.ToLower(args[0])
				if !strings.Contains(strings.ToLower(a.Name), query) &&
					!strings.Contains(strings.ToLower(a.Description), query) {
					continue
				}
			}

			desc := a.Description
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.Name, a.Type, a.Version, desc)
		}

		w.Flush()
		return nil
	},
}

func containsTag(tags []string, tag string) bool {
	tag = strings.ToLower(tag)
	for _, t := range tags {
		if strings.ToLower(t) == tag {
			return true
		}
	}
	return false
}

func init() {
	catalogCmd.Flags().StringVar(&catalogTypeFilter, "type", "", "Filter by artifact type (skill, mcp-server, cli-tool, config, template)")
	catalogCmd.Flags().StringVar(&catalogTagFilter, "tag", "", "Filter by tag")
	rootCmd.AddCommand(catalogCmd)
}
