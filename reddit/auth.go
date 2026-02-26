package reddit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	oauthBaseURL = "https://oauth.reddit.com"
	tokenURL     = "https://www.reddit.com/api/v1/access_token"
	tokenExpiry  = 23 * time.Hour // Reddit tokens last 24h, refresh early
)

// Credentials holds Reddit OAuth app credentials.
type Credentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

// oauthToken holds a bearer token and its expiry.
type oauthToken struct {
	mu        sync.RWMutex
	token     string
	expiresAt time.Time
	creds     Credentials
	http      *http.Client
}

// LoadCredentials reads OAuth credentials.
// Priority: env vars (LURK_CLIENT_ID/LURK_CLIENT_SECRET) > config file (~/.config/lurk/credentials.json).
func LoadCredentials() (Credentials, error) {
	// Check env vars first (useful for MCP server config)
	if id, secret := os.Getenv("LURK_CLIENT_ID"), os.Getenv("LURK_CLIENT_SECRET"); id != "" && secret != "" {
		return Credentials{ClientID: id, ClientSecret: secret}, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return Credentials{}, err
	}
	path := filepath.Join(configDir, "lurk", "credentials.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return Credentials{}, err
	}
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return Credentials{}, err
	}
	return creds, nil
}

// TestCredentials verifies that client ID and secret can obtain an OAuth token.
func TestCredentials(clientID, clientSecret string) error {
	_, err := fetchToken(&http.Client{Timeout: 10 * time.Second}, clientID, clientSecret)
	return err
}

// fetchToken exchanges client credentials for a bearer token.
func fetchToken(httpClient *http.Client, clientID, clientSecret string) (string, error) {
	form := url.Values{"grant_type": {"client_credentials"}}
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error contacting Reddit auth")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read auth response")
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Reddit rejected credentials (HTTP %d)", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("unexpected auth response format")
	}
	if tokenResp.Error != "" {
		return "", fmt.Errorf("auth error: %s", tokenResp.Error)
	}
	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}

	return tokenResp.AccessToken, nil
}

// newOAuthToken creates a token manager that auto-refreshes.
func newOAuthToken(httpClient *http.Client, creds Credentials) *oauthToken {
	return &oauthToken{
		creds: creds,
		http:  httpClient,
	}
}

// get returns a valid bearer token, refreshing if needed.
func (t *oauthToken) get() (string, error) {
	t.mu.RLock()
	if t.token != "" && time.Now().Before(t.expiresAt) {
		tok := t.token
		t.mu.RUnlock()
		return tok, nil
	}
	t.mu.RUnlock()

	t.mu.Lock()
	defer t.mu.Unlock()

	// Double-check after acquiring write lock
	if t.token != "" && time.Now().Before(t.expiresAt) {
		return t.token, nil
	}

	token, err := fetchToken(t.http, t.creds.ClientID, t.creds.ClientSecret)
	if err != nil {
		return "", err
	}

	t.token = token
	t.expiresAt = time.Now().Add(tokenExpiry)
	return token, nil
}
