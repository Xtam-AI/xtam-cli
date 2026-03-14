package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/xtam-ai/xtam-cli/internal/auth"
)

// DownloadAndVerify downloads an artifact archive to a temp file and verifies its SHA256.
func (c *Client) DownloadAndVerify(name, expectedSHA256 string) (string, error) {
	body, _, err := c.DownloadArtifact(name)
	if err != nil {
		return "", err
	}
	defer body.Close()

	// Write to temp file in ~/.xtam/tmp/
	tmpDir := filepath.Join(auth.XtamDir(), "tmp")
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	tmpFile, err := os.CreateTemp(tmpDir, "xtam-download-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Hash while writing
	hasher := sha256.New()
	writer := io.MultiWriter(tmpFile, hasher)

	if _, err := io.Copy(writer, body); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("download interrupted: %w", err)
	}
	tmpFile.Close()

	// Verify SHA256
	actualSHA := hex.EncodeToString(hasher.Sum(nil))
	if actualSHA != expectedSHA256 {
		os.Remove(tmpPath)
		return "", fmt.Errorf("integrity check failed: expected sha256 %s, got %s", expectedSHA256, actualSHA)
	}

	return tmpPath, nil
}
