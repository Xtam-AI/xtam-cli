package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/xtam-ai/xtam-cli/internal/auth"
	"github.com/xtam-ai/xtam-cli/internal/manifest"
)

const defaultBaseURL = "https://registry.xtam.ai"

// Client communicates with the XTAM registry worker.
type Client struct {
	BaseURL    string
	httpClient *http.Client
}

// NewClient creates a registry client.
func NewClient() *Client {
	baseURL := defaultBaseURL
	// Allow override for development
	if env := lookupEnv("XTAM_REGISTRY_URL"); env != "" {
		baseURL = env
	}
	return &Client{
		BaseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func lookupEnv(key string) string {
	return os.Getenv(key)
}

// FetchCatalog retrieves the artifact catalog.
func (c *Client) FetchCatalog() (*manifest.Catalog, error) {
	body, err := c.authenticatedGet("/v1/catalog")
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var catalog manifest.Catalog
	if err := json.NewDecoder(body).Decode(&catalog); err != nil {
		return nil, fmt.Errorf("failed to parse catalog: %w", err)
	}
	return &catalog, nil
}

// FetchManifest retrieves a specific artifact's manifest.
func (c *Client) FetchManifest(name string) (*manifest.Manifest, error) {
	body, err := c.authenticatedGet(fmt.Sprintf("/v1/artifacts/%s", name))
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var m manifest.Manifest
	if err := json.NewDecoder(body).Decode(&m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}
	return &m, nil
}

// DownloadArtifact downloads the artifact archive and returns the response body.
// Caller must close the body.
func (c *Client) DownloadArtifact(name string) (io.ReadCloser, int64, error) {
	token, err := auth.GetValidToken()
	if err != nil {
		return nil, 0, err
	}

	req, err := http.NewRequest("GET", c.BaseURL+fmt.Sprintf("/v1/artifacts/%s/download", name), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "xtam-cli")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("download failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, 0, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	return resp.Body, resp.ContentLength, nil
}

func (c *Client) authenticatedGet(path string) (io.ReadCloser, error) {
	token, err := auth.GetValidToken()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", c.BaseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "xtam-cli")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return resp.Body, nil
	case http.StatusUnauthorized:
		resp.Body.Close()
		return nil, fmt.Errorf("authentication failed — run: xtam login")
	case http.StatusForbidden:
		resp.Body.Close()
		return nil, fmt.Errorf("access denied — only @xtam.ai accounts are allowed")
	case http.StatusNotFound:
		resp.Body.Close()
		return nil, fmt.Errorf("not found")
	default:
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected response: HTTP %d", resp.StatusCode)
	}
}
