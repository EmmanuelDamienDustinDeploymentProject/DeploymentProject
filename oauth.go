package main

import (
	"bytes"
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

// ============================================================================
// Data Structures
// ============================================================================

// TokenInfo stores information about an authenticated token
type TokenInfo struct {
	AccessToken string
	Username    string
	ExpiresAt   time.Time
}

// TokenStore stores valid authentication tokens in memory
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

// oauthState stores OAuth flow state information
type oauthState struct {
	ClientID      string
	RedirectURI   string
	OriginalState string // The state parameter provided by the client
}

// StateStore stores OAuth state parameters during the flow
type StateStore struct {
	sync.RWMutex
	states map[string]*oauthState
}

func (ss *StateStore) Store(state string, os *oauthState) {
	ss.Lock()
	defer ss.Unlock()
	ss.states[state] = os
}

func (ss *StateStore) Get(state string) (*oauthState, bool) {
	ss.RLock()
	defer ss.RUnlock()
	os, exists := ss.states[state]
	return os, exists
}

func (ss *StateStore) Delete(state string) {
	ss.Lock()
	defer ss.Unlock()
	delete(ss.states, state)
}

// RegisteredClient represents a dynamically registered OAuth client
type RegisteredClient struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret,omitempty"`
	RedirectURIs []string `json:"redirect_uris"`
}

// ClientStore stores dynamically registered OAuth clients
type ClientStore struct {
	sync.RWMutex
	clients map[string]*RegisteredClient
}

func (cs *ClientStore) Add(client *RegisteredClient) {
	cs.Lock()
	defer cs.Unlock()
	cs.clients[client.ClientID] = client
}

func (cs *ClientStore) Get(clientID string) (*RegisteredClient, bool) {
	cs.RLock()
	defer cs.RUnlock()
	client, exists := cs.clients[clientID]
	return client, exists
}

// GitHubUser represents a GitHub user
type GitHubUser struct {
	Login     string `json:"login"`
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// ProtectedResourceMetadata represents OAuth 2.0 Protected Resource Metadata (RFC 9728)
type ProtectedResourceMetadata struct {
	Resource               string   `json:"resource"`
	AuthorizationServers   []string `json:"authorization_servers"`
	ScopesSupported        []string `json:"scopes_supported,omitempty"`
	BearerMethodsSupported []string `json:"bearer_methods_supported,omitempty"`
}

// AuthServerMetadata represents OAuth 2.0 Authorization Server Metadata (RFC 8414)
type AuthServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint,omitempty"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	ScopesSupported                   []string `json:"scopes_supported,omitempty"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported,omitempty"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported,omitempty"`
}

// ============================================================================
// Global State
// ============================================================================

var (
	oauthConfig *oauth2.Config
	validTokens = &TokenStore{tokens: make(map[string]*TokenInfo)}
	stateStore  = &StateStore{states: make(map[string]*oauthState)}
	clientStore = &ClientStore{clients: make(map[string]*RegisteredClient)}
)

// ============================================================================
// Initialization
// ============================================================================

// InitializeOAuth sets up the OAuth configuration using environment variables
func InitializeOAuth() {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	clientSecret := os.Getenv("GITHUB_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		log.Println("Warning: GITHUB_CLIENT_ID or GITHUB_CLIENT_SECRET not set, OAuth disabled")
		oauthConfig = nil
		return
	}

	oauthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"read:user", "user:email"},
		Endpoint:     github.Endpoint,
	}

	log.Println("OAuth initialized - all requests require authentication")
}

// ============================================================================
// Token Validation
// ============================================================================

// ValidateBearerToken validates the bearer token from the request
func ValidateBearerToken(token string) error {
	if oauthConfig == nil {
		log.Println("Warning: OAuth not configured, allowing request without authentication")
		return nil
	}

	if token == "" {
		return fmt.Errorf("no bearer token provided")
	}

	// Check if token exists in our store
	info, exists := validTokens.Get(token)
	if exists {
		log.Printf("Request authenticated for user: %s", info.Username)
		return nil
	}

	// Validate with GitHub directly
	ctx := context.Background()
	user, err := getGitHubUser(ctx, token)
	if err != nil {
		return fmt.Errorf("invalid token: %v", err)
	}

	// Store validated token
	expiresAt := time.Now().Add(24 * time.Hour)
	tokenInfo := TokenInfo{
		AccessToken: token,
		Username:    user.Login,
		ExpiresAt:   expiresAt,
	}
	validTokens.Add(token, &tokenInfo)

	log.Printf("Request authenticated for GitHub user: %s", user.Login)
	return nil
}

// bearerTokenMiddleware validates bearer tokens before allowing access
func bearerTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			if oauthConfig == nil {
				log.Println("Warning: OAuth not configured, allowing request without authentication")
				next.ServeHTTP(w, r)
				return
			}

			// Allow discovery methods without auth
			isMCPDiscovery := false
			var requestMethod string
			if r.Method == "POST" && r.Header.Get("Content-Type") == "application/json" {
				bodyBytes, err := io.ReadAll(r.Body)
				if err == nil {
					r.Body.Close()
					r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

					if bytes.Contains(bodyBytes, []byte(`"method":"initialize"`)) {
						isMCPDiscovery = true
						requestMethod = "initialize"
					} else if bytes.Contains(bodyBytes, []byte(`"method":"tools/list"`)) {
						isMCPDiscovery = true
						requestMethod = "tools/list"
					} else if bytes.Contains(bodyBytes, []byte(`"method":"prompts/list"`)) {
						isMCPDiscovery = true
						requestMethod = "prompts/list"
					} else if bytes.Contains(bodyBytes, []byte(`"method":"resources/list"`)) {
						isMCPDiscovery = true
						requestMethod = "resources/list"
					} else {
						if bytes.Contains(bodyBytes, []byte(`"method":`)) {
							log.Printf("Non-discovery MCP request from %s - requires auth", r.RemoteAddr)
						}
					}
				}
			}

			if isMCPDiscovery {
				log.Printf("Allowing discovery method %s without authentication", requestMethod)
				next.ServeHTTP(w, r)
				return
			}

			log.Printf("Authentication required but no token provided from %s", r.RemoteAddr)
			w.Header().Set("WWW-Authenticate", `Bearer realm="mcp-server"`)
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		// Extract token
		token := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}

		if err := ValidateBearerToken(token); err != nil {
			log.Printf("Invalid bearer token from %s: %v", r.RemoteAddr, err)
			w.Header().Set("WWW-Authenticate", `Bearer realm="mcp-server"`)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ============================================================================
// OAuth Metadata Endpoints (RFC 8414, RFC 9728)
// ============================================================================

// oauthMetadataHandler serves the OAuth 2.0 Protected Resource Metadata
func oauthMetadataHandler(w http.ResponseWriter, r *http.Request) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := scheme + "://" + r.Host

	metadata := ProtectedResourceMetadata{
		Resource:               baseURL,
		AuthorizationServers:   []string{baseURL},
		ScopesSupported:        []string{"read:user", "user:email"},
		BearerMethodsSupported: []string{"header"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		log.Printf("Failed to encode OAuth metadata: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// authServerMetadataHandler serves the OAuth 2.0 Authorization Server Metadata
func authServerMetadataHandler(w http.ResponseWriter, r *http.Request) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := scheme + "://" + r.Host

	metadata := AuthServerMetadata{
		Issuer:                            baseURL,
		AuthorizationEndpoint:             baseURL + "/oauth/login",
		TokenEndpoint:                     baseURL + "/oauth/token",
		RegistrationEndpoint:              baseURL + "/dcr",
		ScopesSupported:                   []string{"read:user", "user:email"},
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code"},
		TokenEndpointAuthMethodsSupported: []string{"none"},
		CodeChallengeMethodsSupported:     []string{"S256", "plain"},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		log.Printf("Failed to encode auth server metadata: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// ============================================================================
// Dynamic Client Registration (RFC 7591)
// ============================================================================

// dcrHandler handles dynamic client registration requests
func dcrHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		RedirectURIs []string `json:"redirect_uris"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.RedirectURIs) == 0 {
		http.Error(w, "redirect_uris is required", http.StatusBadRequest)
		return
	}

	// Ensure VS Code's required redirect URIs are included
	vscodeRedirects := []string{
		"http://127.0.0.1:33418",
		"https://vscode.dev/redirect",
	}

	redirectURIs := req.RedirectURIs
	for _, vscodeURI := range vscodeRedirects {
		found := false
		for _, uri := range redirectURIs {
			if uri == vscodeURI {
				found = true
				break
			}
		}
		if !found {
			redirectURIs = append(redirectURIs, vscodeURI)
		}
	}

	// Generate client credentials
	clientID, err := generateRandomString(16)
	if err != nil {
		http.Error(w, "Failed to generate client_id", http.StatusInternalServerError)
		return
	}
	clientSecret, err := generateRandomString(32)
	if err != nil {
		http.Error(w, "Failed to generate client_secret", http.StatusInternalServerError)
		return
	}

	client := &RegisteredClient{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURIs: redirectURIs,
	}

	clientStore.Add(client)

	log.Printf("New client registered: %s with redirect URIs: %v", client.ClientID, client.RedirectURIs)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(client); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

// ============================================================================
// OAuth Flow Handlers
// ============================================================================

// oauthLoginHandler initiates the OAuth flow
func oauthLoginHandler(w http.ResponseWriter, r *http.Request) {
	if oauthConfig == nil {
		http.Error(w, "OAuth not configured", http.StatusInternalServerError)
		return
	}

	clientID := r.URL.Query().Get("client_id")
	if clientID == "" {
		http.Error(w, "client_id is required", http.StatusBadRequest)
		return
	}

	log.Printf("OAuth login request: client_id=%s, redirect_uri=%s", clientID, r.URL.Query().Get("redirect_uri"))

	client, exists := clientStore.Get(clientID)
	if !exists {
		log.Printf("Invalid client_id: %s (not found in store) - auto-registering with provided redirect_uri", clientID)

		// Auto-register the client
		redirectURI := r.URL.Query().Get("redirect_uri")
		if redirectURI == "" {
			http.Error(w, "redirect_uri is required for client registration", http.StatusBadRequest)
			return
		}

		client = &RegisteredClient{
			ClientID:     clientID,
			ClientSecret: "",
			RedirectURIs: []string{redirectURI, "http://127.0.0.1:33418", "https://vscode.dev/redirect"},
		}
		clientStore.Add(client)
		log.Printf("Auto-registered client %s with redirect_uri: %s", clientID, redirectURI)
	}

	redirectURI := r.URL.Query().Get("redirect_uri")
	if redirectURI == "" {
		http.Error(w, "redirect_uri is required", http.StatusBadRequest)
		return
	}

	// Normalize redirect URIs by removing trailing slashes
	normalizeURI := func(uri string) string {
		if len(uri) > 0 && uri[len(uri)-1] == '/' {
			return uri[:len(uri)-1]
		}
		return uri
	}

	validRedirect := false
	normalizedRequestURI := normalizeURI(redirectURI)
	for _, uri := range client.RedirectURIs {
		if normalizeURI(uri) == normalizedRequestURI {
			validRedirect = true
			break
		}
	}
	if !validRedirect {
		log.Printf("Invalid redirect_uri: %s (not in allowed list: %v)", redirectURI, client.RedirectURIs)
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}

	// Get the original state from the client
	clientState := r.URL.Query().Get("state")

	// Generate our own internal state for the GitHub OAuth flow
	internalState := generateStateToken()

	// Store mapping
	stateStore.Store(internalState, &oauthState{
		ClientID:      clientID,
		RedirectURI:   redirectURI,
		OriginalState: clientState,
	})

	// Build GitHub OAuth URL
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	serverCallback := fmt.Sprintf("%s://%s/oauth/callback", scheme, r.Host)

	config := &oauth2.Config{
		ClientID:     oauthConfig.ClientID,
		ClientSecret: oauthConfig.ClientSecret,
		RedirectURL:  serverCallback,
		Scopes:       []string{"read:user", "user:email"},
		Endpoint:     github.Endpoint,
	}

	url := config.AuthCodeURL(internalState, oauth2.AccessTypeOnline)

	log.Printf("Redirecting to GitHub OAuth for client %s, will return to %s", clientID, redirectURI)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// oauthCallbackHandler handles the callback from GitHub, exchanges the code for a token,
// and shows a success page.
func oauthCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if oauthConfig == nil {
		http.Error(w, "OAuth not configured", http.StatusInternalServerError)
		return
	}

	// Validate state
	state := r.URL.Query().Get("state")
	oauthStateData, exists := stateStore.Get(state)
	if !exists {
		http.Error(w, "Invalid state parameter. Please try logging in again.", http.StatusBadRequest)
		return
	}
	stateStore.Delete(state) // State is single-use

	// Exchange authorization code for a token
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Authorization code not found.", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	serverCallback := fmt.Sprintf("%s://%s/oauth/callback", scheme, r.Host)
	config := &oauth2.Config{
		ClientID:     oauthConfig.ClientID,
		ClientSecret: oauthConfig.ClientSecret,
		RedirectURL:  serverCallback,
		Scopes:       []string{"read:user", "user:email"},
		Endpoint:     github.Endpoint,
	}

	token, err := config.Exchange(ctx, code)
	if err != nil {
		log.Printf("Failed to exchange token: %v", err)
		http.Error(w, "Failed to exchange authorization code for a token.", http.StatusInternalServerError)
		return
	}

	// Get user info from GitHub
	user, err := getGitHubUser(ctx, token.AccessToken)
	if err != nil {
		log.Printf("Failed to get user info: %v", err)
		http.Error(w, "Failed to retrieve user information from GitHub.", http.StatusInternalServerError)
		return
	}

	// Store the token internally
	expiresAt := time.Now().Add(24 * time.Hour)
	if !token.Expiry.IsZero() && token.Expiry.After(time.Now()) {
		expiresAt = token.Expiry
	}
	tokenInfo := TokenInfo{
		AccessToken: token.AccessToken,
		Username:    user.Login,
		ExpiresAt:   expiresAt,
	}
	validTokens.Add(token.AccessToken, &tokenInfo)

	log.Printf("Successfully authenticated and stored token for user %s", user.Login)

	// Instead of redirecting with the code, we now need to get the token to the client.
	// The original client (e.g., MCP Inspector) is polling the token endpoint.
	// We need to store the token in a way that the token handler can retrieve it.
	// We can use the original client state for this.
	stateStore.Store(oauthStateData.OriginalState, &oauthState{
		// We'll store the access token here temporarily.
		// This is a simplification. In a real-world scenario, you'd have a more secure way
		// to associate the token with the client's session.
		ClientID: token.AccessToken,
	})

	// Now, redirect to the original client's redirect_uri, but without the code.
	// The client will then proceed to the token endpoint.
	finalRedirectURL := fmt.Sprintf("%s?state=%s", oauthStateData.RedirectURI, oauthStateData.OriginalState)
	http.Redirect(w, r, finalRedirectURL, http.StatusTemporaryRedirect)
}

// oauthTokenHandler handles token exchange requests
func oauthTokenHandler(w http.ResponseWriter, r *http.Request) {
	if oauthConfig == nil {
		http.Error(w, "OAuth not configured", http.StatusInternalServerError)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	grantType := r.FormValue("grant_type")
	if grantType != "authorization_code" {
		http.Error(w, "Unsupported grant_type", http.StatusBadRequest)
		return
	}

	// The client provides the original state. We use this to look up the token
	// that the callback handler stored.
	clientState := r.FormValue("state")
	storedState, exists := stateStore.Get(clientState)
	if !exists {
		http.Error(w, "Invalid or expired state.", http.StatusBadRequest)
		return
	}
	stateStore.Delete(clientState) // It's single-use

	accessToken := storedState.ClientID // We repurposed this field to hold the token
	tokenInfo, exists := validTokens.Get(accessToken)
	if !exists {
		http.Error(w, "Invalid or expired token.", http.StatusInternalServerError)
		return
	}

	log.Printf("Token issued for user %s via state lookup", tokenInfo.Username)

	// Return token response
	response := map[string]interface{}{
		"access_token": tokenInfo.AccessToken,
		"token_type":   "Bearer",
		"expires_in":   int(time.Until(tokenInfo.ExpiresAt).Seconds()),
		"scope":        "read:user user:email",
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// Helper Functions
// ============================================================================

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

// generateStateToken generates a random state token for CSRF protection
func generateStateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// generateRandomString generates a URL-safe, base64 encoded random string
func generateRandomString(s int) (string, error) {
	b := make([]byte, s)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
