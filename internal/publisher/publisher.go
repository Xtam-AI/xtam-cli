package publisher

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xtam-ai/xtam-cli/internal/auth"
	"github.com/xtam-ai/xtam-cli/internal/manifest"
)

const (
	githubAPI  = "https://api.github.com"
	repoOwner  = "Xtam-AI"
	repoName   = "registry"
)

// PublishOpts holds the options for publishing an artifact.
type PublishOpts struct {
	SourceDir   string
	Name        string
	Type        manifest.ArtifactType
	Version     string
	Description string
	Tags        []string
	AuthorName  string
	AuthorEmail string
}

// Publish packages a directory and pushes it to the registry repo.
func Publish(opts *PublishOpts) error {
	token, err := loadPublishToken()
	if err != nil {
		return err
	}

	archiveName := fmt.Sprintf("%s-%s.tar.gz", opts.Name, opts.Version)

	// Step 1: Package directory into tar.gz
	fmt.Printf("Packaging %s...\n", opts.SourceDir)
	archiveBytes, err := createTarGz(opts.SourceDir, opts.Name)
	if err != nil {
		return fmt.Errorf("failed to package: %w", err)
	}
	fmt.Printf("  Archive: %s (%s)\n", archiveName, formatBytes(int64(len(archiveBytes))))

	// Step 2: Compute SHA256
	hash := sha256.Sum256(archiveBytes)
	sha := hex.EncodeToString(hash[:])
	fmt.Printf("  SHA256:  %s\n", sha)

	// Step 3: Build manifest
	m := &manifest.Manifest{
		Name:        opts.Name,
		Version:     opts.Version,
		Type:        opts.Type,
		Description: opts.Description,
		Author: manifest.Author{
			Name:  opts.AuthorName,
			Email: opts.AuthorEmail,
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Archive:   archiveName,
		SHA256:    sha,
		SizeBytes: int64(len(archiveBytes)),
		Install:   buildInstall(opts),
		Requires:  &manifest.Requires{CLIVersion: ">=0.1.0"},
	}

	manifestBytes, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Step 4: Push archive to GitHub
	fmt.Printf("\nPushing to registry...\n")
	archivePath := fmt.Sprintf("artifacts/%s/%s", opts.Name, archiveName)
	err = pushFile(token, archivePath, archiveBytes, fmt.Sprintf("Publish %s v%s (archive)", opts.Name, opts.Version))
	if err != nil {
		return fmt.Errorf("failed to push archive: %w", err)
	}
	fmt.Printf("  Uploaded %s\n", archivePath)

	// Step 5: Push manifest to GitHub
	manifestPath := fmt.Sprintf("artifacts/%s/manifest.json", opts.Name)
	err = pushFile(token, manifestPath, manifestBytes, fmt.Sprintf("Publish %s v%s (manifest)", opts.Name, opts.Version))
	if err != nil {
		return fmt.Errorf("failed to push manifest: %w", err)
	}
	fmt.Printf("  Uploaded %s\n", manifestPath)

	// Step 6: Update catalog.json
	fmt.Printf("  Updating catalog.json...\n")
	err = updateCatalog(token, opts)
	if err != nil {
		return fmt.Errorf("failed to update catalog: %w", err)
	}

	fmt.Printf("\nPublished %s v%s to the XTAM registry.\n", opts.Name, opts.Version)
	return nil
}

func buildInstall(opts *PublishOpts) manifest.Install {
	install := manifest.Install{Target: opts.Type}

	switch opts.Type {
	case manifest.TypeSkill:
		install.SkillName = opts.Name
		install.Scope = "global"
	case manifest.TypeMCPServer:
		install.ServerName = opts.Name
		install.Scope = "project"
	case manifest.TypeCLITool:
		install.BinaryName = opts.Name
	case manifest.TypeConfig:
		install.Scope = "project"
	case manifest.TypeTemplate:
		install.ExtractTo = "."
	}

	return install
}

// createTarGz creates a tar.gz archive from a directory.
// The archive contains files under a top-level directory named after the artifact.
func createTarGz(sourceDir, topLevelName string) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	sourceDir = filepath.Clean(sourceDir)

	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files/dirs (except the source dir itself)
		base := filepath.Base(path)
		if base != "." && strings.HasPrefix(base, ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Build archive path with top-level directory
		archivePath := filepath.Join(topLevelName, relPath)
		if relPath == "." {
			archivePath = topLevelName
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = archivePath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})

	if err != nil {
		return nil, err
	}

	tw.Close()
	gw.Close()

	return buf.Bytes(), nil
}

// pushFile creates or updates a file in the registry repo via the GitHub Contents API.
func pushFile(token, path string, content []byte, message string) error {
	// Check if file exists (to get its SHA for updates)
	existingSHA := getFileSHA(token, path)

	body := map[string]interface{}{
		"message": message,
		"content": base64.StdEncoding.EncodeToString(content),
	}
	if existingSHA != "" {
		body["sha"] = existingSHA
	}

	bodyBytes, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", githubAPI, repoOwner, repoName, path)

	req, err := http.NewRequest("PUT", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "xtam-cli")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// getFileSHA returns the SHA of a file in the repo, or empty string if not found.
func getFileSHA(token, path string) string {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", githubAPI, repoOwner, repoName, path)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "xtam-cli")

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		SHA string `json:"sha"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.SHA
}

// updateCatalog reads catalog.json, adds/updates the entry, and pushes it back.
func updateCatalog(token string, opts *PublishOpts) error {
	// Fetch current catalog
	url := fmt.Sprintf("%s/repos/%s/%s/contents/catalog.json", githubAPI, repoOwner, repoName)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "xtam-cli")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var fileResp struct {
		Content string `json:"content"`
		SHA     string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&fileResp); err != nil {
		return err
	}

	// Decode base64 content
	catalogBytes, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(fileResp.Content, "\n", ""))
	if err != nil {
		return fmt.Errorf("failed to decode catalog: %w", err)
	}

	var catalog manifest.Catalog
	if err := json.Unmarshal(catalogBytes, &catalog); err != nil {
		return fmt.Errorf("failed to parse catalog: %w", err)
	}

	// Add or update entry
	newEntry := manifest.CatalogEntry{
		Name:        opts.Name,
		Type:        opts.Type,
		Version:     opts.Version,
		Description: opts.Description,
		Tags:        opts.Tags,
	}

	found := false
	for i, a := range catalog.Artifacts {
		if a.Name == opts.Name {
			catalog.Artifacts[i] = newEntry
			found = true
			break
		}
	}
	if !found {
		catalog.Artifacts = append(catalog.Artifacts, newEntry)
	}

	catalog.UpdatedAt = time.Now().UTC()

	// Push updated catalog
	updatedBytes, _ := json.MarshalIndent(catalog, "", "  ")

	pushBody := map[string]interface{}{
		"message": fmt.Sprintf("Update catalog: %s v%s", opts.Name, opts.Version),
		"content": base64.StdEncoding.EncodeToString(updatedBytes),
		"sha":     fileResp.SHA,
	}
	pushBodyBytes, _ := json.Marshal(pushBody)

	putReq, _ := http.NewRequest("PUT", url, bytes.NewReader(pushBodyBytes))
	putReq.Header.Set("Authorization", "Bearer "+token)
	putReq.Header.Set("Accept", "application/vnd.github+json")
	putReq.Header.Set("User-Agent", "xtam-cli")

	putResp, err := http.DefaultClient.Do(putReq)
	if err != nil {
		return err
	}
	defer putResp.Body.Close()

	if putResp.StatusCode != 200 && putResp.StatusCode != 201 {
		respBody, _ := io.ReadAll(putResp.Body)
		return fmt.Errorf("failed to update catalog: %d %s", putResp.StatusCode, string(respBody))
	}

	return nil
}

// Token management for publishing

func publishTokenPath() string {
	return filepath.Join(auth.XtamDir(), "publish.json")
}

type publishAuth struct {
	Token string `json:"token"`
}

func loadPublishToken() (string, error) {
	// Check env var first
	if token := os.Getenv("XTAM_GITHUB_TOKEN"); token != "" {
		return token, nil
	}

	data, err := os.ReadFile(publishTokenPath())
	if err != nil {
		return "", fmt.Errorf("no publish token configured. Run: xtam publish-setup")
	}

	var pa publishAuth
	if err := json.Unmarshal(data, &pa); err != nil {
		return "", fmt.Errorf("corrupt publish config. Run: xtam publish-setup")
	}

	return pa.Token, nil
}

// SavePublishToken stores a GitHub PAT for publishing.
func SavePublishToken(token string) error {
	dir := filepath.Dir(publishTokenPath())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, _ := json.MarshalIndent(publishAuth{Token: token}, "", "  ")
	return os.WriteFile(publishTokenPath(), data, 0600)
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
