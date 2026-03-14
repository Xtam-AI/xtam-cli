package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const (
	googleAuthEndpoint  = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenEndpoint = "https://oauth2.googleapis.com/token"
	scopes              = "openid email profile"
	requiredDomain      = "xtam.ai"
)

// ClientID and ClientSecret are set at build time via ldflags,
// or overridden by env vars.
var (
	ClientID     = "PLACEHOLDER.apps.googleusercontent.com"
	ClientSecret = "PLACEHOLDER"
)

type TokenResponse struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Error        string `json:"error,omitempty"`
	ErrorDesc    string `json:"error_description,omitempty"`
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

// Login performs the Google OAuth localhost redirect flow.
// Opens the browser, starts a temporary local server to capture the callback,
// exchanges the auth code for tokens, and verifies the @xtam.ai domain.
func Login(ctx context.Context) (*TokenResponse, *IDTokenClaims, error) {
	clientID := getClientID()
	clientSecret := getClientSecret()

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	// Generate state for CSRF protection
	state, err := randomString(32)
	if err != nil {
		listener.Close()
		return nil, nil, fmt.Errorf("failed to generate state: %w", err)
	}

	// Build Google OAuth URL
	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&access_type=offline&prompt=consent&hd=%s",
		googleAuthEndpoint,
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(scopes),
		url.QueryEscape(state),
		url.QueryEscape(requiredDomain),
	)

	// Channel to receive the auth code
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	// Start local HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Verify state
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("OAuth state mismatch — possible CSRF attack")
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html><body><h2>Authentication failed</h2><p>State mismatch. Please try again.</p></body></html>")
			return
		}

		// Check for error
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			errCh <- fmt.Errorf("OAuth error: %s", errParam)
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "<html><body><h2>Authentication failed</h2><p>%s</p></body></html>", errParam)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no authorization code received")
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, "<html><body><h2>Authentication failed</h2><p>No code received.</p></body></html>")
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body style="font-family: system-ui; text-align: center; padding: 60px;">
			<h2>Authenticated!</h2>
			<p>You can close this tab and return to the terminal.</p>
		</body></html>`)

		codeCh <- code
	})

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer server.Close()

	// Open browser
	fmt.Printf("\n  Opening browser to authenticate...\n")
	fmt.Printf("  If the browser doesn't open, visit:\n\n    %s\n\n", authURL)
	openBrowser(authURL)

	// Wait for callback
	var code string
	select {
	case code = <-codeCh:
		// got it
	case err := <-errCh:
		return nil, nil, err
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}

	fmt.Println("  Exchanging code for token...")

	// Exchange code for tokens
	tr, err := exchangeCode(clientID, clientSecret, code, redirectURI)
	if err != nil {
		return nil, nil, err
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

func exchangeCode(clientID, clientSecret, code, redirectURI string) (*TokenResponse, error) {
	resp, err := http.PostForm(googleTokenEndpoint, url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	})
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}
	if tr.Error != "" {
		return nil, fmt.Errorf("token error: %s (%s)", tr.Error, tr.ErrorDesc)
	}

	return &tr, nil
}

// ParseIDToken decodes the JWT payload without signature verification.
// Signature verification is done server-side by the Cloudflare Worker.
func ParseIDToken(idToken string) (*IDTokenClaims, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT format")
	}

	payload := parts[1]
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
	clientSecret := getClientSecret()
	resp, err := http.PostForm(googleTokenEndpoint, url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
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
	if id := os.Getenv("XTAM_OAUTH_CLIENT_ID"); id != "" {
		return id
	}
	return ClientID
}

func getClientSecret() string {
	if s := os.Getenv("XTAM_OAUTH_CLIENT_SECRET"); s != "" {
		return s
	}
	return ClientSecret
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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
