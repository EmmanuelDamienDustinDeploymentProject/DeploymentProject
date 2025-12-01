// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"io"
	"net/http"
	"net/url"
	"strings"
)

// TokenProxyHandler proxies token requests to GitHub to avoid CORS issues
type TokenProxyHandler struct {
	config *Config
}

// NewTokenProxyHandler creates a new token proxy handler
func NewTokenProxyHandler(config *Config) *TokenProxyHandler {
	return &TokenProxyHandler{
		config: config,
	}
}

// ServeHTTP implements http.Handler
func (h *TokenProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Forward the request to GitHub's token endpoint
	formData := url.Values{}
	formData.Set("client_id", h.config.GitHubClientID)
	formData.Set("client_secret", h.config.GitHubClientSecret)
	formData.Set("code", r.FormValue("code"))
	formData.Set("redirect_uri", r.FormValue("redirect_uri"))
	formData.Set("code_verifier", r.FormValue("code_verifier"))
	formData.Set("grant_type", r.FormValue("grant_type"))

	// Create request to GitHub
	req, err := http.NewRequest("POST", h.config.GitHubTokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	// Send request to GitHub
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	// Forward GitHub's response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

// AuthorizeProxyHandler proxies authorization requests to GitHub
type AuthorizeProxyHandler struct {
	config *Config
}

// NewAuthorizeProxyHandler creates a new authorize proxy handler
func NewAuthorizeProxyHandler(config *Config) *AuthorizeProxyHandler {
	return &AuthorizeProxyHandler{
		config: config,
	}
}

// ServeHTTP implements http.Handler
func (h *AuthorizeProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Build GitHub authorization URL with query parameters
	authURL, err := url.Parse(h.config.GitHubAuthURL)
	if err != nil {
		http.Error(w, "Invalid authorization URL", http.StatusInternalServerError)
		return
	}

	// Copy query parameters from the request
	query := authURL.Query()
	for key, values := range r.URL.Query() {
		for _, value := range values {
			query.Add(key, value)
		}
	}

	// Ensure client_id is set
	if query.Get("client_id") == "" {
		query.Set("client_id", h.config.GitHubClientID)
	}

	authURL.RawQuery = query.Encode()

	// Redirect to GitHub
	http.Redirect(w, r, authURL.String(), http.StatusFound)
}
