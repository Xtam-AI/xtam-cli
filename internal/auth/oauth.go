package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	googleDeviceEndpoint = "https://oauth2.googleapis.com/device/code"
	googleTokenEndpoint  = "https://oauth2.googleapis.com/token"
	scopes               = "openid email profile"
	requiredDomain       = "xtam.ai"
)

// ClientID is set at build time via ldflags, or overridden by XTAM_OAUTH_CLIENT_ID env var.
var ClientID = "PLACEHOLDER.apps.googleusercontent.com"

type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type TokenResponse struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Error        string `json:"error,omitempty"`
}

type IDTokenClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	HD            string `json:"hd"`
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	Exp           int64  `json:"exp"`
	Aud           string `json:"aud"`
	Iss           string `json:"iss"`
}

// Login performs the Google OAuth device authorization flow.
// It returns the token response only if the user's email is @xtam.ai.
func Login(ctx context.Context) (*TokenResponse, *IDTokenClaims, error) {
	clientID := getClientID()

	// Step 1: Request device code
	resp, err := http.PostForm(googleDeviceEndpoint, url.Values{
		"client_id": {clientID},
		"scope":     {scopes},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("device code request failed: %w", err)
	}
	defer resp.Body.Close()

	var dcr DeviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&dcr); err != nil {
		return nil, nil, fmt.Errorf("failed to parse device code response: %w", err)
	}

	// Step 2: Show instructions to user
	fmt.Printf("\n  Open this URL in your browser:\n\n    %s\n\n", dcr.VerificationURL)
	fmt.Printf("  Enter code: %s\n\n", dcr.UserCode)
	fmt.Println("  Waiting for authentication...")

	openBrowser(dcr.VerificationURL)

	// Step 3: Poll for token
	interval := time.Duration(dcr.Interval) * time.Second
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	deadline := time.Now().Add(time.Duration(dcr.ExpiresIn) * time.Second)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(interval):
		}

		tr, err := pollToken(clientID, dcr.DeviceCode)
		if err != nil {
			return nil, nil, err
		}
		if tr == nil {
			continue // authorization_pending
		}

		// Verify the id_token domain
		claims, err := ParseIDToken(tr.IDToken)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid id_token: %w", err)
		}

		if claims.HD != requiredDomain {
			return nil, nil, fmt.Errorf("access denied: email must be @%s (got %s)", requiredDomain, claims.Email)
		}
		if !claims.EmailVerified {
			return nil, nil, fmt.Errorf("access denied: email %s is not verified", claims.Email)
		}

		return tr, claims, nil
	}

	return nil, nil, fmt.Errorf("authentication timed out after %d seconds", dcr.ExpiresIn)
}

func pollToken(clientID, deviceCode string) (*TokenResponse, error) {
	resp, err := http.PostForm(googleTokenEndpoint, url.Values{
		"client_id":   {clientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	})
	if err != nil {
		return nil, nil // retry on network error
	}
	defer resp.Body.Close()

	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, nil
	}

	switch tr.Error {
	case "":
		return &tr, nil
	case "authorization_pending":
		return nil, nil
	case "slow_down":
		return nil, nil
	default:
		return nil, fmt.Errorf("oauth error: %s", tr.Error)
	}
}

// ParseIDToken decodes the JWT payload without signature verification.
// Signature verification is done server-side by the Cloudflare Worker.
func ParseIDToken(idToken string) (*IDTokenClaims, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	payload := parts[1]
	// Add padding if needed
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	var claims IDTokenClaims
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse JWT claims: %w", err)
	}

	return &claims, nil
}

// RefreshIDToken uses a refresh token to get a new id_token.
func RefreshIDToken(refreshToken string) (*TokenResponse, error) {
	clientID := getClientID()
	resp, err := http.PostForm(googleTokenEndpoint, url.Values{
		"client_id":     {clientID},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	})
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}
	if tr.Error != "" {
		return nil, fmt.Errorf("refresh error: %s", tr.Error)
	}

	return &tr, nil
}

func getClientID() string {
	// Allow override via environment for development
	if id := lookupEnv("XTAM_OAUTH_CLIENT_ID"); id != "" {
		return id
	}
	return ClientID
}

func lookupEnv(key string) string {
	return os.Getenv(key)
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("open", url).Start()
	case "linux":
		_ = exec.Command("xdg-open", url).Start()
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}
}
