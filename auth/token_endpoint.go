package auth

// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// TokenEndpointHandler handles OAuth 2.1 token requests
type TokenEndpointHandler struct {
	config        *Config
	clientStorage ClientStorage
	tokenStorage  TokenStorage
}

// NewTokenEndpointHandler creates a new token endpoint handler
func NewTokenEndpointHandler(config *Config, clientStorage ClientStorage, tokenStorage TokenStorage) *TokenEndpointHandler {
	return &TokenEndpointHandler{
		config:        config,
		clientStorage: clientStorage,
		tokenStorage:  tokenStorage,
	}
}

// ServeHTTP implements http.Handler
func (h *TokenEndpointHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		h.sendError(w, "invalid_request", "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		h.sendError(w, "invalid_request", "Invalid form data", http.StatusBadRequest)
		return
	}

	grantType := r.FormValue("grant_type")
	if grantType != "authorization_code" {
		h.sendError(w, "unsupported_grant_type", "Only authorization_code grant type is supported", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	if code == "" {
		h.sendError(w, "invalid_request", "code is required", http.StatusBadRequest)
		return
	}

	clientID := r.FormValue("client_id")
	if clientID == "" {
		h.sendError(w, "invalid_request", "client_id is required", http.StatusBadRequest)
		return
	}

	codeVerifier := r.FormValue("code_verifier")
	if codeVerifier == "" {
		h.sendError(w, "invalid_request", "code_verifier is required (PKCE)", http.StatusBadRequest)
		return
	}

	redirectURI := r.FormValue("redirect_uri")
	if redirectURI == "" {
		h.sendError(w, "invalid_request", "redirect_uri is required", http.StatusBadRequest)
		return
	}

	// Validate client
	client, err := h.clientStorage.GetClient(clientID)
	if err != nil || client == nil {
		log.Printf("Unknown client_id in token request: %s", clientID)
		h.sendError(w, "invalid_client", "Unknown client_id", http.StatusUnauthorized)
		return
	}

	// Retrieve auth code info
	authCodeInfo, err := h.tokenStorage.GetAuthCode(code)
	if err != nil {
		log.Printf("Invalid or expired authorization code")
		h.sendError(w, "invalid_grant", "Invalid or expired authorization code", http.StatusBadRequest)
		return
	}

	// Verify client_id matches
	if authCodeInfo.ClientID != clientID {
		log.Printf("client_id mismatch: expected %s, got %s", authCodeInfo.ClientID, clientID)
		h.sendError(w, "invalid_grant", "client_id mismatch", http.StatusBadRequest)
		return
	}

	// Verify redirect_uri matches
	if authCodeInfo.RedirectURI != redirectURI {
		log.Printf("redirect_uri mismatch: expected %s, got %s", authCodeInfo.RedirectURI, redirectURI)
		h.sendError(w, "invalid_grant", "redirect_uri mismatch", http.StatusBadRequest)
		return
	}

	// Verify PKCE code_verifier
	if !verifyPKCE(codeVerifier, authCodeInfo.CodeChallenge, authCodeInfo.CodeChallengeMethod) {
		log.Printf("PKCE verification failed")
		h.sendError(w, "invalid_grant", "PKCE verification failed", http.StatusBadRequest)
		return
	}

	// Delete the authorization code (one-time use)
	if err := h.tokenStorage.DeleteAuthCode(code); err != nil {
		log.Printf("Failed to delete auth code: %v", err)
	}

	// Generate access token
	accessToken, err := generateRandomString(43) // 43 bytes = ~256 bits
	if err != nil {
		log.Printf("Failed to generate access token: %v", err)
		h.sendError(w, "server_error", "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	// Store access token
	expiresAt := time.Now().Add(h.config.TokenExpiryDuration)
	tokenInfo := &AccessTokenInfo{
		ClientID:          clientID,
		Scope:             authCodeInfo.Scope,
		Resource:          authCodeInfo.Resource,
		GitHubAccessToken: authCodeInfo.GitHubAccessToken,
		ExpiresAt:         expiresAt,
		CreatedAt:         time.Now(),
	}

	if err := h.tokenStorage.StoreAccessToken(accessToken, tokenInfo); err != nil {
		log.Printf("Failed to store access token: %v", err)
		h.sendError(w, "server_error", "Failed to store access token", http.StatusInternalServerError)
		return
	}

	// Return token response
	response := map[string]interface{}{
		"access_token": accessToken,
		"token_type":   "Bearer",
		"expires_in":   int(h.config.TokenExpiryDuration.Seconds()),
		"scope":        authCodeInfo.Scope,
	}

	if authCodeInfo.Resource != "" {
		response["resource"] = authCodeInfo.Resource
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode token response: %v", err)
	}
}

// sendError sends an OAuth error response
func (h *TokenEndpointHandler) sendError(w http.ResponseWriter, errorCode, errorDescription string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]string{
		"error":             errorCode,
		"error_description": errorDescription,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}

// verifyPKCE verifies the PKCE code_verifier against the code_challenge
func verifyPKCE(codeVerifier, codeChallenge, method string) bool {
	if method != "S256" {
		return false
	}

	// Compute SHA256 hash of code_verifier
	hash := sha256.Sum256([]byte(codeVerifier))
	computed := base64.RawURLEncoding.EncodeToString(hash[:])

	return computed == codeChallenge
}
