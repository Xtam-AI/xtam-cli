package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/xtam-ai/xtam-cli/internal/manifest"
)

type CLIToolInstaller struct{}

func (i *CLIToolInstaller) Install(m *manifest.Manifest, archivePath string) (string, error) {
	binaryName := m.Install.BinaryName
	if binaryName == "" {
		binaryName = m.Name
	}

	// Determine platform
	platform := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	if m.Install.Platforms != nil {
		if _, ok := m.Install.Platforms[platform]; !ok {
			return "", fmt.Errorf("no binary available for platform %s", platform)
		}
	}

	// Extract to temp dir first
	tmpDir, err := os.MkdirTemp("", "xtam-clitool-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := extractTarGz(archivePath, tmpDir); err != nil {
		return "", fmt.Errorf("failed to extract: %w", err)
	}

	// Find the binary in the extracted files
	binaryPath := filepath.Join(tmpDir, binaryName)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Try looking for it without the top-level strip (flat archive)
		entries, _ := os.ReadDir(tmpDir)
		for _, e := range entries {
			if !e.IsDir() {
				binaryPath = filepath.Join(tmpDir, e.Name())
				break
			}
		}
	}

	// Install to ~/.local/bin/
	home, _ := os.UserHomeDir()
	installDir := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create %s: %w", installDir, err)
	}

	destPath := filepath.Join(installDir, binaryName)

	// Read the binary
	data, err := os.ReadFile(binaryPath)
	if err != nil {
		return "", fmt.Errorf("binary not found in archive: %w", err)
	}

	// Write with execute permission
	if err := os.WriteFile(destPath, data, 0755); err != nil {
		return "", fmt.Errorf("failed to install binary: %w", err)
	}

	return destPath, nil
}

func (i *CLIToolInstaller) Uninstall(m *manifest.Manifest) error {
	binaryName := m.Install.BinaryName
	if binaryName == "" {
		binaryName = m.Name
	}

	home, _ := os.UserHomeDir()
	destPath := filepath.Join(home, ".local", "bin", binaryName)
	err := os.Remove(destPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
