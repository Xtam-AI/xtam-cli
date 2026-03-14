package installer

import (
	"fmt"

	"github.com/xtam-ai/xtam-cli/internal/manifest"
)

// Installer handles installing/uninstalling a specific artifact type.
type Installer interface {
	Install(m *manifest.Manifest, archivePath string) (installPath string, err error)
	Uninstall(m *manifest.Manifest) error
}

// ForType returns the appropriate installer for an artifact type.
func ForType(t manifest.ArtifactType) (Installer, error) {
	switch t {
	case manifest.TypeSkill:
		return &SkillInstaller{}, nil
	case manifest.TypeMCPServer:
		return &MCPInstaller{}, nil
	case manifest.TypeCLITool:
		return &CLIToolInstaller{}, nil
	case manifest.TypeConfig:
		return &ConfigInstaller{}, nil
	case manifest.TypeTemplate:
		return &TemplateInstaller{}, nil
	default:
		return nil, fmt.Errorf("unknown artifact type: %s", t)
	}
}
