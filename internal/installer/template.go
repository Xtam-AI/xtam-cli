package installer

import (
	"fmt"
	"os"

	"github.com/xtam-ai/xtam-cli/internal/manifest"
)

type TemplateInstaller struct{}

func (i *TemplateInstaller) Install(m *manifest.Manifest, archivePath string) (string, error) {
	extractTo := m.Install.ExtractTo
	if extractTo == "" {
		extractTo = "."
	}

	if err := os.MkdirAll(extractTo, 0755); err != nil {
		return "", fmt.Errorf("failed to create target directory: %w", err)
	}

	if err := extractTarGz(archivePath, extractTo); err != nil {
		return "", fmt.Errorf("failed to extract template: %w", err)
	}

	if m.Install.PostInstall != "" {
		fmt.Printf("  Post-install: %s\n", m.Install.PostInstall)
	}

	absPath, _ := os.Getwd()
	return absPath, nil
}

func (i *TemplateInstaller) Uninstall(m *manifest.Manifest) error {
	fmt.Printf("  Note: template files must be removed manually\n")
	return nil
}
