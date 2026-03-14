package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xtam-ai/xtam-cli/internal/auth"
	"github.com/xtam-ai/xtam-cli/internal/manifest"
)

// InstalledArtifact tracks an installed artifact.
type InstalledArtifact struct {
	Name        string                `json:"name"`
	Version     string                `json:"version"`
	Type        manifest.ArtifactType `json:"type"`
	InstalledAt time.Time             `json:"installed_at"`
	InstallPath string                `json:"install_path"`
	SHA256      string                `json:"sha256"`
}

// InstalledState is the full installed.json file.
type InstalledState struct {
	Artifacts map[string]*InstalledArtifact `json:"artifacts"`
}

func stateFilePath() string {
	return filepath.Join(auth.XtamDir(), "installed.json")
}

// Load reads the installed state from disk.
func Load() (*InstalledState, error) {
	data, err := os.ReadFile(stateFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return &InstalledState{Artifacts: make(map[string]*InstalledArtifact)}, nil
		}
		return nil, fmt.Errorf("failed to read state: %w", err)
	}

	var state InstalledState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("corrupt state file: %w", err)
	}
	if state.Artifacts == nil {
		state.Artifacts = make(map[string]*InstalledArtifact)
	}
	return &state, nil
}

// Save writes the installed state to disk.
func (s *InstalledState) Save() error {
	dir := filepath.Dir(stateFilePath())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create state dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	return os.WriteFile(stateFilePath(), data, 0644)
}

// Record marks an artifact as installed.
func (s *InstalledState) Record(m *manifest.Manifest, installPath string) {
	s.Artifacts[m.Name] = &InstalledArtifact{
		Name:        m.Name,
		Version:     m.Version,
		Type:        m.Type,
		InstalledAt: time.Now(),
		InstallPath: installPath,
		SHA256:      m.SHA256,
	}
}

// Remove marks an artifact as uninstalled.
func (s *InstalledState) Remove(name string) {
	delete(s.Artifacts, name)
}

// Get returns info about an installed artifact, or nil if not installed.
func (s *InstalledState) Get(name string) *InstalledArtifact {
	return s.Artifacts[name]
}

// List returns all installed artifacts.
func (s *InstalledState) List() []*InstalledArtifact {
	result := make([]*InstalledArtifact, 0, len(s.Artifacts))
	for _, a := range s.Artifacts {
		result = append(result, a)
	}
	return result
}
