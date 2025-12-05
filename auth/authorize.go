package auth

// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AuthorizationHandler handles OAuth 2.1 authorization requests
// This is the proper authorization endpoint for DCR clients
type AuthorizationHandler struct {
	config        *Config
	clientStorage ClientStorage
	stateStore    *StateStore // Store for OAuth state and PKCE parameters
}

// StateStore stores OAuth state, PKCE parameters, and client info during the flow
type StateStore struct {
	states map[string]*AuthState
}

// AuthState holds the state for an ongoing authorization flow
type AuthState struct {
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	CodeChallenge       string
	CodeChallengeMethod string
	Resource            string
	CreatedAt           time.Time
}

// NewStateStore creates a new state store
func NewStateStore() *StateStore {
	return &StateStore{
		states: make(map[string]*AuthState),
	}
}

// Store saves an auth state
func (s *StateStore) Store(state string, authState *AuthState) {
	s.states[state] = authState
	// Clean up old states (older than 10 minutes)
	cutoff := time.Now().Add(-10 * time.Minute)
	for k, v := range s.states {
		if v.CreatedAt.Before(cutoff) {
			delete(s.states, k)
		}
	}
}

// Get retrieves an auth state
func (s *StateStore) Get(state string) (*AuthState, bool) {
	authState, ok := s.states[state]
	return authState, ok
}

// Delete removes an auth state
func (s *StateStore) Delete(state string) {
	delete(s.states, state)
}

// NewAuthorizationHandler creates a new authorization handler
func NewAuthorizationHandler(config *Config, clientStorage ClientStorage) *AuthorizationHandler {
	return &AuthorizationHandler{
		config:        config,
		clientStorage: clientStorage,
		stateStore:    NewStateStore(),
	}
}

// GetStateStore returns the state store (needed by callback handler)
func (h *AuthorizationHandler) GetStateStore() *StateStore {
	return h.stateStore
}

// ServeHTTP implements http.Handler
func (h *AuthorizationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()

	responseType := query.Get("response_type")
	clientID := query.Get("client_id")
	redirectURI := query.Get("redirect_uri")
	scope := query.Get("scope")
	clientState := query.Get("state")
	codeChallenge := query.Get("code_challenge")
	codeChallengeMethod := query.Get("code_challenge_method")
	resource := query.Get("resource")

	// Validate response_type
	if responseType != "code" {
		h.sendError(w, r, redirectURI, clientState, "unsupported_response_type", "Only 'code' response type is supported")
		return
	}

	// Validate client_id
	if clientID == "" {
		h.sendError(w, r, redirectURI, clientState, "invalid_request", "client_id is required")
		return
	}

	// Look up client (for DCR clients)
	client, err := h.clientStorage.GetClient(clientID)
	if err != nil || client == nil {
		log.Printf("Unknown client_id: %s", clientID)
		h.sendError(w, r, redirectURI, clientState, "invalid_client", "Unknown client_id")
		return
	}

	// Validate redirect_uri
	if redirectURI == "" {
		h.sendError(w, r, redirectURI, clientState, "invalid_request", "redirect_uri is required")
		return
	}

	// Check if redirect_uri is registered for this client
	validRedirect := false
	for _, uri := range client.Metadata.RedirectURIs {
		if uri == redirectURI {
			validRedirect = true
			break
		}
	}
	if !validRedirect {
		log.Printf("Invalid redirect_uri %s for client %s", redirectURI, clientID)
		h.sendError(w, r, "", clientState, "invalid_request", "redirect_uri not registered for this client")
		return
	}

	// Validate PKCE (required for OAuth 2.1)
	if codeChallenge == "" {
		h.sendError(w, r, redirectURI, clientState, "invalid_request", "code_challenge is required (PKCE)")
		return
	}
	if codeChallengeMethod != "S256" {
		h.sendError(w, r, redirectURI, clientState, "invalid_request", "code_challenge_method must be S256")
		return
	}

	// Validate scope
	if scope == "" {
		scope = "mcp:tools mcp:resources read:user"
	}
	requestedScopes := strings.Split(scope, " ")
	for _, s := range requestedScopes {
		if !h.config.IsScopeSupported(s) {
			h.sendError(w, r, redirectURI, clientState, "invalid_scope", fmt.Sprintf("Scope '%s' is not supported", s))
			return
		}
	}

	// Generate internal state for GitHub OAuth flow
	internalState, err := generateRandomString(32)
	if err != nil {
		log.Printf("Failed to generate state: %v", err)
		h.sendError(w, r, redirectURI, clientState, "server_error", "Failed to generate state")
		return
	}

	// Store the authorization state
	authState := &AuthState{
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		Scope:               scope,
		State:               clientState,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		Resource:            resource,
		CreatedAt:           time.Now(),
	}
	h.stateStore.Store(internalState, authState)

	// Build GitHub authorization URL
	githubAuthURL, err := url.Parse(h.config.GitHubAuthURL)
	if err != nil {
		log.Printf("Invalid GitHub auth URL: %v", err)
		h.sendError(w, r, redirectURI, clientState, "server_error", "Invalid authorization server configuration")
		return
	}

	// Set up GitHub OAuth parameters
	githubQuery := githubAuthURL.Query()
	githubQuery.Set("client_id", h.config.GitHubClientID)
	githubQuery.Set("redirect_uri", h.config.ServerURL+"/oauth/callback")
	githubQuery.Set("scope", "read:user")
	githubQuery.Set("state", internalState)
	githubAuthURL.RawQuery = githubQuery.Encode()

	// Redirect user to GitHub for authentication
	http.Redirect(w, r, githubAuthURL.String(), http.StatusFound)
}

// sendError sends an OAuth error response
func (h *AuthorizationHandler) sendError(w http.ResponseWriter, r *http.Request, redirectURI, state, errorCode, errorDescription string) {
	if redirectURI == "" {
		// Can't redirect, return error directly
		http.Error(w, fmt.Sprintf("%s: %s", errorCode, errorDescription), http.StatusBadRequest)
		return
	}

	// Build error redirect URL
	errorURL, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "Invalid redirect_uri", http.StatusBadRequest)
		return
	}

	query := errorURL.Query()
	query.Set("error", errorCode)
	query.Set("error_description", errorDescription)
	if state != "" {
		query.Set("state", state)
	}
	errorURL.RawQuery = query.Encode()

	http.Redirect(w, r, errorURL.String(), http.StatusFound)
}

// generateRandomString generates a random base64-encoded string
func generateRandomString(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
