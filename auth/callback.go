// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CallbackHandler handles OAuth callbacks from GitHub
type CallbackHandler struct {
	config       *Config
	stateStore   *StateStore
	tokenStorage TokenStorage
}

// TokenStorage stores authorization codes and access tokens
type TokenStorage interface {
	StoreAuthCode(code string, authInfo *AuthCodeInfo) error
	GetAuthCode(code string) (*AuthCodeInfo, error)
	DeleteAuthCode(code string) error
	StoreAccessToken(token string, tokenInfo *AccessTokenInfo) error
	GetAccessToken(token string) (*AccessTokenInfo, error)
}

// AuthCodeInfo holds information about an authorization code
type AuthCodeInfo struct {
	ClientID            string
	RedirectURI         string
	Scope               string
	CodeChallenge       string
	CodeChallengeMethod string
	Resource            string
	GitHubAccessToken   string // The token we got from GitHub
	ExpiresAt           time.Time
	CreatedAt           time.Time
}

// AccessTokenInfo holds information about an access token
type AccessTokenInfo struct {
	ClientID          string
	Scope             string
	Resource          string
	GitHubAccessToken string
	ExpiresAt         time.Time
	CreatedAt         time.Time
}

// InMemoryTokenStorage is an in-memory implementation of TokenStorage
type InMemoryTokenStorage struct {
	authCodes    map[string]*AuthCodeInfo
	accessTokens map[string]*AccessTokenInfo
}

// NewInMemoryTokenStorage creates a new in-memory token storage
func NewInMemoryTokenStorage() *InMemoryTokenStorage {
	return &InMemoryTokenStorage{
		authCodes:    make(map[string]*AuthCodeInfo),
		accessTokens: make(map[string]*AccessTokenInfo),
	}
}

func (s *InMemoryTokenStorage) StoreAuthCode(code string, authInfo *AuthCodeInfo) error {
	s.authCodes[code] = authInfo
	// Clean up expired codes
	now := time.Now()
	for k, v := range s.authCodes {
		if v.ExpiresAt.Before(now) {
			delete(s.authCodes, k)
		}
	}
	return nil
}

func (s *InMemoryTokenStorage) GetAuthCode(code string) (*AuthCodeInfo, error) {
	authInfo, ok := s.authCodes[code]
	if !ok {
		return nil, fmt.Errorf("authorization code not found")
	}
	if time.Now().After(authInfo.ExpiresAt) {
		delete(s.authCodes, code)
		return nil, fmt.Errorf("authorization code expired")
	}
	return authInfo, nil
}

func (s *InMemoryTokenStorage) DeleteAuthCode(code string) error {
	delete(s.authCodes, code)
	return nil
}

func (s *InMemoryTokenStorage) StoreAccessToken(token string, tokenInfo *AccessTokenInfo) error {
	s.accessTokens[token] = tokenInfo
	// Clean up expired tokens
	now := time.Now()
	for k, v := range s.accessTokens {
		if v.ExpiresAt.Before(now) {
			delete(s.accessTokens, k)
		}
	}
	return nil
}

func (s *InMemoryTokenStorage) GetAccessToken(token string) (*AccessTokenInfo, error) {
	tokenInfo, ok := s.accessTokens[token]
	if !ok {
		return nil, fmt.Errorf("access token not found")
	}
	if time.Now().After(tokenInfo.ExpiresAt) {
		delete(s.accessTokens, token)
		return nil, fmt.Errorf("access token expired")
	}
	return tokenInfo, nil
}

// NewCallbackHandler creates a new callback handler
func NewCallbackHandler(config *Config, stateStore *StateStore, tokenStorage TokenStorage) *CallbackHandler {
	return &CallbackHandler{
		config:       config,
		stateStore:   stateStore,
		tokenStorage: tokenStorage,
	}
}

// ServeHTTP implements http.Handler
func (h *CallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get the authorization code and state from the query parameters
	githubCode := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")
	errorDescription := r.URL.Query().Get("error_description")

	// Check for errors from GitHub
	if errorParam != "" {
		http.Error(w, fmt.Sprintf("Authorization error: %s - %s", errorParam, errorDescription), http.StatusBadRequest)
		return
	}

	// Check if we have a code
	if githubCode == "" {
		http.Error(w, "No authorization code received", http.StatusBadRequest)
		return
	}

	// Retrieve the auth state
	authState, ok := h.stateStore.Get(state)
	if !ok {
		http.Error(w, "Invalid or expired state parameter", http.StatusBadRequest)
		return
	}

	// Exchange GitHub code for access token
	githubToken, err := h.exchangeGitHubCode(githubCode)
	if err != nil {
		log.Printf("Failed to exchange GitHub code: %v", err)
		h.sendErrorRedirect(w, r, authState, "server_error", "Failed to obtain access token")
		return
	}

	// Generate our own authorization code for the client
	ourAuthCode, err := generateRandomString(32)
	if err != nil {
		log.Printf("Failed to generate auth code: %v", err)
		h.sendErrorRedirect(w, r, authState, "server_error", "Failed to generate authorization code")
		return
	}

	// Store the authorization code with the GitHub token
	authCodeInfo := &AuthCodeInfo{
		ClientID:            authState.ClientID,
		RedirectURI:         authState.RedirectURI,
		Scope:               authState.Scope,
		CodeChallenge:       authState.CodeChallenge,
		CodeChallengeMethod: authState.CodeChallengeMethod,
		Resource:            authState.Resource,
		GitHubAccessToken:   githubToken,
		ExpiresAt:           time.Now().Add(10 * time.Minute), // Auth codes expire in 10 minutes
		CreatedAt:           time.Now(),
	}

	if err := h.tokenStorage.StoreAuthCode(ourAuthCode, authCodeInfo); err != nil {
		log.Printf("Failed to store auth code: %v", err)
		h.sendErrorRedirect(w, r, authState, "server_error", "Failed to store authorization code")
		return
	}

	// Clean up state
	h.stateStore.Delete(state)

	// Redirect back to the client with our authorization code
	redirectURL, err := url.Parse(authState.RedirectURI)
	if err != nil {
		http.Error(w, "Invalid redirect URI", http.StatusBadRequest)
		return
	}

	query := redirectURL.Query()
	query.Set("code", ourAuthCode)
	if authState.State != "" {
		query.Set("state", authState.State)
	}
	redirectURL.RawQuery = query.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

// exchangeGitHubCode exchanges a GitHub authorization code for an access token
func (h *CallbackHandler) exchangeGitHubCode(code string) (string, error) {
	// Build token request
	data := url.Values{}
	data.Set("client_id", h.config.GitHubClientID)
	data.Set("client_secret", h.config.GitHubClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", h.config.ServerURL+"/oauth/callback")

	// Make request to GitHub
	req, err := http.NewRequest("POST", h.config.GitHubTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to exchange code: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub token exchange failed: %s - %s", resp.Status, string(body))
	}

	// Parse response
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.Error != "" {
		return "", fmt.Errorf("GitHub error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}

	return tokenResp.AccessToken, nil
}

// sendErrorRedirect redirects back to the client with an error
func (h *CallbackHandler) sendErrorRedirect(w http.ResponseWriter, r *http.Request, authState *AuthState, errorCode, errorDescription string) {
	redirectURL, err := url.Parse(authState.RedirectURI)
	if err != nil {
		http.Error(w, "Invalid redirect URI", http.StatusBadRequest)
		return
	}

	query := redirectURL.Query()
	query.Set("error", errorCode)
	query.Set("error_description", errorDescription)
	if authState.State != "" {
		query.Set("state", authState.State)
	}
	redirectURL.RawQuery = query.Encode()

	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}
