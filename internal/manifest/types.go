package manifest

import "time"

type ArtifactType string

const (
	TypeSkill     ArtifactType = "skill"
	TypeMCPServer ArtifactType = "mcp-server"
	TypeCLITool   ArtifactType = "cli-tool"
	TypeConfig    ArtifactType = "config"
	TypeTemplate  ArtifactType = "template"
)

// Catalog is the master index returned by /v1/catalog.
type Catalog struct {
	Registry  string         `json:"registry"`
	UpdatedAt time.Time      `json:"updated_at"`
	Artifacts []CatalogEntry `json:"artifacts"`
}

type CatalogEntry struct {
	Name        string       `json:"name"`
	Type        ArtifactType `json:"type"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Tags        []string     `json:"tags,omitempty"`
}

// Manifest is the per-artifact metadata from manifest.json.
type Manifest struct {
	Name        string       `json:"name"`
	Version     string       `json:"version"`
	Type        ArtifactType `json:"type"`
	Description string       `json:"description"`
	Author      Author       `json:"author"`
	CreatedAt   time.Time    `json:"created_at,omitempty"`
	UpdatedAt   time.Time    `json:"updated_at,omitempty"`
	Archive     string       `json:"archive"`
	SHA256      string       `json:"sha256"`
	SizeBytes   int64        `json:"size_bytes,omitempty"`
	Install     Install      `json:"install"`
	Requires    *Requires    `json:"requires,omitempty"`
	Changelog   string       `json:"changelog,omitempty"`
}

type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Install struct {
	Target ArtifactType `json:"target"`

	// Skill-specific
	SkillName string `json:"skill_name,omitempty"`
	Scope     string `json:"scope,omitempty"` // "project" or "global"

	// MCP-specific
	ServerName string     `json:"server_name,omitempty"`
	MCPConfig  *MCPConfig `json:"mcp_config,omitempty"`

	// CLI tool-specific
	BinaryName string            `json:"binary_name,omitempty"`
	Platforms  map[string]string `json:"platforms,omitempty"`

	// Config-specific
	Files []ConfigFile `json:"files,omitempty"`

	// Template-specific
	ExtractTo   string `json:"extract_to,omitempty"`
	PostInstall string `json:"post_install,omitempty"`
}

type MCPConfig struct {
	Type    string            `json:"type"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	URL     string            `json:"url,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

type ConfigFile struct {
	Source        string `json:"source"`
	Destination   string `json:"destination"`
	Scope         string `json:"scope"`
	MergeStrategy string `json:"merge_strategy"`
}

type Requires struct {
	CLIVersion   string   `json:"cli_version,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}
