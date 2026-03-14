package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StoredAuth is persisted to ~/.xtam/auth.json.
type StoredAuth struct {
	IDToken      string    `json:"id_token"`
	RefreshToken string    `json:"refresh_token"`
	Email        string    `json:"email"`
	Name         string    `json:"name,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// XtamDir returns ~/.xtam/
func XtamDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".xtam")
}

// AuthFilePath returns ~/.xtam/auth.json
func AuthFilePath() string {
	return filepath.Join(XtamDir(), "auth.json")
}

// SaveAuth writes auth state to disk with owner-only permissions.
func SaveAuth(auth *StoredAuth) error {
	dir := filepath.Dir(AuthFilePath())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth: %w", err)
	}
	return os.WriteFile(AuthFilePath(), data, 0600)
}

// LoadAuth reads auth state from disk.
func LoadAuth() (*StoredAuth, error) {
	data, err := os.ReadFile(AuthFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not logged in — run: xtam login")
		}
		return nil, fmt.Errorf("failed to read auth file: %w", err)
	}
	var auth StoredAuth
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, fmt.Errorf("corrupt auth file — run: xtam login")
	}
	return &auth, nil
}

// DeleteAuth removes the auth file.
func DeleteAuth() error {
	err := os.Remove(AuthFilePath())
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove auth file: %w", err)
	}
	return nil
}

// GetValidToken returns a valid id_token, refreshing if expired.
func GetValidToken() (string, error) {
	auth, err := LoadAuth()
	if err != nil {
		return "", err
	}

	// If token is still valid (with 60s buffer), return it
	if time.Now().Before(auth.ExpiresAt.Add(-60 * time.Second)) {
		return auth.IDToken, nil
	}

	// Try to refresh
	if auth.RefreshToken == "" {
		return "", fmt.Errorf("session expired — run: xtam login")
	}

	tr, err := RefreshIDToken(auth.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("session expired — run: xtam login")
	}

	claims, err := ParseIDToken(tr.IDToken)
	if err != nil {
		return "", fmt.Errorf("invalid refreshed token — run: xtam login")
	}

	auth.IDToken = tr.IDToken
	auth.ExpiresAt = time.Unix(claims.Exp, 0)
	if tr.RefreshToken != "" {
		auth.RefreshToken = tr.RefreshToken
	}

	if err := SaveAuth(auth); err != nil {
		// Non-fatal: token still works for this request
		fmt.Fprintf(os.Stderr, "warning: could not save refreshed token: %v\n", err)
	}

	return auth.IDToken, nil
}
