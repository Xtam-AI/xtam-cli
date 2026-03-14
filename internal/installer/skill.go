package installer

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/xtam-ai/xtam-cli/internal/manifest"
)

type SkillInstaller struct{}

func (s *SkillInstaller) Install(m *manifest.Manifest, archivePath string) (string, error) {
	skillName := m.Install.SkillName
	if skillName == "" {
		skillName = m.Name
	}

	// Determine target: global (~/.claude/skills/) or project (.claude/skills/)
	var targetDir string
	if m.Install.Scope == "project" {
		targetDir = filepath.Join(".claude", "skills", skillName)
	} else {
		home, _ := os.UserHomeDir()
		targetDir = filepath.Join(home, ".claude", "skills", skillName)
	}

	// Remove existing
	os.RemoveAll(targetDir)

	// Extract tar.gz
	if err := extractTarGz(archivePath, targetDir); err != nil {
		return "", fmt.Errorf("failed to extract skill: %w", err)
	}

	return targetDir, nil
}

func (s *SkillInstaller) Uninstall(m *manifest.Manifest) error {
	skillName := m.Install.SkillName
	if skillName == "" {
		skillName = m.Name
	}

	home, _ := os.UserHomeDir()
	targetDir := filepath.Join(home, ".claude", "skills", skillName)
	return os.RemoveAll(targetDir)
}

// extractTarGz extracts a .tar.gz archive, stripping the top-level directory.
func extractTarGz(archivePath, targetDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("not a valid gzip archive: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading archive: %w", err)
		}

		// Strip the top-level directory
		name := header.Name
		parts := strings.SplitN(name, "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			continue
		}
		relPath := parts[1]

		outPath := filepath.Join(targetDir, relPath)

		// Prevent path traversal
		if !strings.HasPrefix(filepath.Clean(outPath), filepath.Clean(targetDir)) {
			return fmt.Errorf("archive contains path traversal: %s", name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(outPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}

	return nil
}
