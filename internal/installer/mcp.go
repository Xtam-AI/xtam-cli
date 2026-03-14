package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/xtam-ai/xtam-cli/internal/auth"
	"github.com/xtam-ai/xtam-cli/internal/manifest"
)

type MCPInstaller struct{}

func (i *MCPInstaller) Install(m *manifest.Manifest, archivePath string) (string, error) {
	serverName := m.Install.ServerName
	if serverName == "" {
		serverName = m.Name
	}

	// Extract server files to ~/.xtam/mcp-servers/<name>/
	serverDir := filepath.Join(auth.XtamDir(), "mcp-servers", serverName)
	os.RemoveAll(serverDir)
	if err := extractTarGz(archivePath, serverDir); err != nil {
		return "", fmt.Errorf("failed to extract MCP server: %w", err)
	}

	// Merge MCP config into the appropriate settings file
	if m.Install.MCPConfig != nil {
		if err := mergeMCPConfig(serverName, m.Install.MCPConfig, m.Install.Scope, serverDir); err != nil {
			return "", fmt.Errorf("failed to configure MCP server: %w", err)
		}
	}

	return serverDir, nil
}

func (i *MCPInstaller) Uninstall(m *manifest.Manifest) error {
	serverName := m.Install.ServerName
	if serverName == "" {
		serverName = m.Name
	}

	// Remove server files
	serverDir := filepath.Join(auth.XtamDir(), "mcp-servers", serverName)
	os.RemoveAll(serverDir)

	// Remove from MCP config
	scope := m.Install.Scope
	if scope == "" {
		scope = "project"
	}

	configPath := mcpConfigPath(scope)
	existing := readJSONFile(configPath)
	if servers, ok := existing["mcpServers"].(map[string]interface{}); ok {
		delete(servers, serverName)
		existing["mcpServers"] = servers
		writeJSONFile(configPath, existing)
	}

	return nil
}

func mergeMCPConfig(serverName string, cfg *manifest.MCPConfig, scope, serverDir string) error {
	if scope == "" {
		scope = "project"
	}

	configPath := mcpConfigPath(scope)

	existing := readJSONFile(configPath)

	// Build the server entry
	entry := map[string]interface{}{
		"type": cfg.Type,
	}
	if cfg.Command != "" {
		// Expand ~/ references to the actual server directory
		cmd := cfg.Command
		entry["command"] = cmd
		if len(cfg.Args) > 0 {
			args := make([]interface{}, len(cfg.Args))
			for i, a := range cfg.Args {
				// Replace placeholder with actual install path
				if a == "${XTAM_MCP_DIR}" {
					a = serverDir
				}
				args[i] = a
			}
			entry["args"] = args
		}
	}
	if cfg.URL != "" {
		entry["url"] = cfg.URL
	}
	if len(cfg.Env) > 0 {
		entry["env"] = cfg.Env
	}

	// Merge into mcpServers
	servers, ok := existing["mcpServers"].(map[string]interface{})
	if !ok {
		servers = make(map[string]interface{})
	}
	servers[serverName] = entry
	existing["mcpServers"] = servers

	return writeJSONFile(configPath, existing)
}

func mcpConfigPath(scope string) string {
	if scope == "global" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".claude", "settings.json")
	}
	return ".mcp.json"
}

func readJSONFile(path string) map[string]interface{} {
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]interface{})
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return make(map[string]interface{})
	}
	return result
}

func writeJSONFile(path string, data map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0644)
}
