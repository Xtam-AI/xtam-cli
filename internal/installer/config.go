package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xtam-ai/xtam-cli/internal/manifest"
)

type ConfigInstaller struct{}

func (i *ConfigInstaller) Install(m *manifest.Manifest, archivePath string) (string, error) {
	// Extract to temp dir
	tmpDir, err := os.MkdirTemp("", "xtam-config-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := extractTarGz(archivePath, tmpDir); err != nil {
		return "", fmt.Errorf("failed to extract: %w", err)
	}

	// Process each file in the install spec
	for _, cf := range m.Install.Files {
		srcPath := filepath.Join(tmpDir, cf.Source)
		destPath := expandPath(cf.Destination, cf.Scope)

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return "", fmt.Errorf("failed to create directory for %s: %w", destPath, err)
		}

		switch cf.MergeStrategy {
		case "replace":
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return "", fmt.Errorf("source file %s not found in archive: %w", cf.Source, err)
			}
			if err := os.WriteFile(destPath, data, 0644); err != nil {
				return "", err
			}

		case "append":
			srcData, err := os.ReadFile(srcPath)
			if err != nil {
				return "", fmt.Errorf("source file %s not found: %w", cf.Source, err)
			}
			f, err := os.OpenFile(destPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return "", err
			}
			_, err = f.Write(append([]byte("\n"), srcData...))
			f.Close()
			if err != nil {
				return "", err
			}

		case "deep_merge":
			if err := deepMergeJSON(srcPath, destPath); err != nil {
				return "", fmt.Errorf("failed to merge %s: %w", cf.Destination, err)
			}

		default:
			return "", fmt.Errorf("unknown merge strategy: %s", cf.MergeStrategy)
		}
	}

	return "~/.claude/ (config files)", nil
}

func (i *ConfigInstaller) Uninstall(m *manifest.Manifest) error {
	// Config uninstall is best-effort — we can't easily undo merges/appends
	fmt.Printf("  Note: config artifacts may require manual cleanup\n")
	return nil
}

func expandPath(path, scope string) string {
	if scope == "project" || scope == "" {
		return path
	}
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return filepath.Join(home, path)
}

func deepMergeJSON(srcPath, destPath string) error {
	srcData, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	var src map[string]interface{}
	if err := json.Unmarshal(srcData, &src); err != nil {
		return fmt.Errorf("source is not valid JSON: %w", err)
	}

	dest := make(map[string]interface{})
	if destData, err := os.ReadFile(destPath); err == nil {
		json.Unmarshal(destData, &dest)
	}

	merged := merge(dest, src)

	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(destPath, append(out, '\n'), 0644)
}

func merge(base, overlay map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range base {
		result[k] = v
	}
	for k, v := range overlay {
		if baseVal, exists := result[k]; exists {
			baseMap, baseIsMap := baseVal.(map[string]interface{})
			overlayMap, overlayIsMap := v.(map[string]interface{})
			if baseIsMap && overlayIsMap {
				result[k] = merge(baseMap, overlayMap)
				continue
			}
		}
		result[k] = v
	}
	return result
}
