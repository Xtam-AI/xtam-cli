package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xtam-ai/xtam-cli/internal/manifest"
	"github.com/xtam-ai/xtam-cli/internal/publisher"
)

var (
	publishType        string
	publishVersion     string
	publishName        string
	publishDescription string
	publishTags        []string
	publishAuthorName  string
	publishAuthorEmail string
)

var publishCmd = &cobra.Command{
	Use:   "publish <directory>",
	Short: "Package and publish an artifact to the XTAM registry",
	Long: `Packages a directory into a tar.gz archive, generates a manifest,
updates the catalog, and pushes everything to the registry.

Examples:
  xtam publish ./my-skill --type skill --version 1.0.0
  xtam publish ./my-mcp-server --type mcp-server --version 0.2.0 --tag jira --tag atlassian
  xtam publish ./my-tool --type cli-tool --version 1.0.0 --description "Audit data extractor"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceDir := args[0]

		// Validate source directory exists
		info, err := os.Stat(sourceDir)
		if err != nil || !info.IsDir() {
			return fmt.Errorf("'%s' is not a directory", sourceDir)
		}

		// Validate required flags
		if publishType == "" {
			return fmt.Errorf("--type is required (skill, mcp-server, cli-tool, config, template)")
		}
		if publishVersion == "" {
			return fmt.Errorf("--version is required (e.g., 1.0.0)")
		}

		// Validate type
		artType := manifest.ArtifactType(publishType)
		switch artType {
		case manifest.TypeSkill, manifest.TypeMCPServer, manifest.TypeCLITool,
			manifest.TypeConfig, manifest.TypeTemplate:
			// valid
		default:
			return fmt.Errorf("invalid type '%s' — must be: skill, mcp-server, cli-tool, config, template", publishType)
		}

		// Default name to directory name
		name := publishName
		if name == "" {
			name = filepath.Base(filepath.Clean(sourceDir))
		}

		// Default description
		desc := publishDescription
		if desc == "" {
			desc = fmt.Sprintf("%s %s artifact", name, publishType)
		}

		// Default author
		authorName := publishAuthorName
		authorEmail := publishAuthorEmail
		if authorName == "" {
			authorName = "XTAM"
		}
		if authorEmail == "" {
			authorEmail = "eng@xtam.ai"
		}

		opts := &publisher.PublishOpts{
			SourceDir:   sourceDir,
			Name:        name,
			Type:        artType,
			Version:     publishVersion,
			Description: desc,
			Tags:        publishTags,
			AuthorName:  authorName,
			AuthorEmail: authorEmail,
		}

		return publisher.Publish(opts)
	},
}

func init() {
	publishCmd.Flags().StringVar(&publishType, "type", "", "Artifact type: skill, mcp-server, cli-tool, config, template (required)")
	publishCmd.Flags().StringVar(&publishVersion, "version", "", "Semantic version, e.g. 1.0.0 (required)")
	publishCmd.Flags().StringVar(&publishName, "name", "", "Artifact name (defaults to directory name)")
	publishCmd.Flags().StringVar(&publishDescription, "description", "", "Short description")
	publishCmd.Flags().StringSliceVar(&publishTags, "tag", nil, "Tags (repeatable)")
	publishCmd.Flags().StringVar(&publishAuthorName, "author", "", "Author name")
	publishCmd.Flags().StringVar(&publishAuthorEmail, "email", "", "Author email")
	rootCmd.AddCommand(publishCmd)
}
