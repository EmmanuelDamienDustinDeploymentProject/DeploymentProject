package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

var (
	oauthConfig *oauth2.Config
	// Store valid tokens in memory (in production, use Redis or a database)
	validTokens = &TokenStore{
		tokens: make(map[string]*TokenInfo),
	}
)

type TokenInfo struct {
	AccessToken string
	Username    string
	ExpiresAt   time.Time
}

type TokenStore struct {
	sync.RWMutex
	tokens map[string]*TokenInfo
}

func (ts *TokenStore) Add(token string, info *TokenInfo) {
	ts.Lock()
	defer ts.Unlock()
	ts.tokens[token] = info
}

func (ts *TokenStore) Get(token string) (*TokenInfo, bool) {
	ts.RLock()
	defer ts.RUnlock()
	info, exists := ts.tokens[token]
	if !exists {
		return nil, false
	}
	// Check if token is expired
	if time.Now().After(info.ExpiresAt) {
		return nil, false
	}
	return info, true
}

func (ts *TokenStore) Delete(token string) {
	ts.Lock()
	defer ts.Unlock()
	delete(ts.tokens, token)
}

// InitOAuth initializes the OAuth2 configuration
func InitOAuth() {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")
	redirectURL := os.Getenv("OAUTH_REDIRECT_URL")

	if clientID == "" || clientSecret == "" {
		log.Println("Warning: GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET not set. OAuth will not work.")
		log.Println("Set these environment variables to enable GitHub OAuth.")
		return
	}

	if redirectURL == "" {
		redirectURL = "http://localhost:8080/oauth/callback"
	}

	oauthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"read:user", "user:email"},
		Endpoint:     github.Endpoint,
	}

	log.Printf("OAuth initialized with redirect URL: %s", redirectURL)
}

// generateStateToken generates a random state token for CSRF protection
func generateStateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// oauthLoginHandler initiates the OAuth flow
func oauthLoginHandler(w http.ResponseWriter, r *http.Request) {
	if oauthConfig == nil {
		http.Error(w, "OAuth not configured", http.StatusInternalServerError)
		return
	}

	state := generateStateToken()

	// In production, store state in a session or Redis with expiration
	// For now, we'll pass it through the flow
	url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline)

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// GitHubUser represents the GitHub user info
type GitHubUser struct {
	Login     string `json:"login"`
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// getGitHubUser fetches the authenticated user's info from GitHub
func getGitHubUser(ctx context.Context, accessToken string) (*GitHubUser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github API returned status %d: %s", resp.StatusCode, string(body))
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// oauthCallbackHandler handles the OAuth callback
func oauthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if oauthConfig == nil {
		http.Error(w, "OAuth not configured", http.StatusInternalServerError)
		return
	}

	// Get the authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No code in request", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	ctx := context.Background()
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		log.Printf("Failed to exchange token: %v", err)
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	// Get user info from GitHub
	user, err := getGitHubUser(ctx, token.AccessToken)
	if err != nil {
		log.Printf("Failed to get user info: %v", err)
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	// Store the token
	expiresAt := time.Now().Add(24 * time.Hour) // Tokens valid for 24 hours
	if token.Expiry.After(time.Now()) {
		expiresAt = token.Expiry
	}

	validTokens.Add(token.AccessToken, &TokenInfo{
		AccessToken: token.AccessToken,
		Username:    user.Login,
		ExpiresAt:   expiresAt,
	})

	log.Printf("User %s authenticated successfully", user.Login)

	// Return HTML page with the token
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>OAuth Success</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 600px;
            margin: 50px auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        h1 { color: #24292e; }
        .token {
            background: #f6f8fa;
            padding: 15px;
            border-radius: 6px;
            font-family: monospace;
            word-break: break-all;
            margin: 20px 0;
        }
        .copy-btn {
            background: #2ea44f;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 6px;
            cursor: pointer;
            font-size: 14px;
        }
        .copy-btn:hover { background: #2c974b; }
        .info { color: #586069; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <h1>✓ Authentication Successful</h1>
        <p>Welcome, <strong>%s</strong>!</p>
        <p class="info">Use this bearer token to authenticate your MCP client:</p>
        <div class="token" id="token">%s</div>
        <button class="copy-btn" onclick="copyToken()">Copy Token</button>
        <p class="info" style="margin-top: 20px;">
            Token expires: %s<br>
            Add this token to your MCP client configuration.
        </p>
    </div>
    <script>
        function copyToken() {
            const token = document.getElementById('token').textContent;
            navigator.clipboard.writeText(token).then(() => {
                const btn = document.querySelector('.copy-btn');
                btn.textContent = '✓ Copied!';
                setTimeout(() => btn.textContent = 'Copy Token', 2000);
            });
        }
    </script>
</body>
</html>
`, user.Login, token.AccessToken, expiresAt.Format(time.RFC1123))

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// ValidateBearerToken validates the bearer token from the request
func ValidateBearerToken(token string) error {
	if oauthConfig == nil {
		// If OAuth is not configured, allow all requests (development mode)
		log.Println("Warning: OAuth not configured, allowing request without authentication")
		return nil
	}

	if token == "" {
		return fmt.Errorf("no bearer token provided")
	}

	// Check if token exists in our store
	info, exists := validTokens.Get(token)
	if !exists {
		return fmt.Errorf("invalid or expired token")
	}

	log.Printf("Request authenticated for user: %s", info.Username)
	return nil
}

// CreateOAuthHTTPClient creates an HTTP client with OAuth token for MCP client
func CreateOAuthHTTPClient(token string) *http.Client {
	return &http.Client{
		Transport: &tokenTransport{
			token: token,
			base:  http.DefaultTransport,
		},
	}
}

// tokenTransport adds the bearer token to all requests
type tokenTransport struct {
	token string
	base  http.RoundTripper
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req)
}
