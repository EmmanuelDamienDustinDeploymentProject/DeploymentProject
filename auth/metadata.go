// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"encoding/json"
	"net/http"
)

// ProtectedResourceMetadataHandler handles requests for OAuth 2.0 Protected Resource Metadata
// Serves the /.well-known/oauth-protected-resource endpoint per RFC 9728
type ProtectedResourceMetadataHandler struct {
	config *Config
}

// NewProtectedResourceMetadataHandler creates a new handler for protected resource metadata
func NewProtectedResourceMetadataHandler(config *Config) *ProtectedResourceMetadataHandler {
	return &ProtectedResourceMetadataHandler{
		config: config,
	}
}

// ServeHTTP implements http.Handler
func (h *ProtectedResourceMetadataHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Build the metadata response
	metadata := ProtectedResourceMetadata{
		Resource: h.config.ServerURL,
		AuthorizationServers: []string{
			h.config.ServerURL, // Point to our server's auth metadata endpoint
		},
		ScopesSupported: h.config.ScopesSupported,
		BearerMethodsSupported: []string{
			"header", // We only support Authorization header
		},
		ResourceDocumentation: h.config.ServerURL + "/docs",
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	// Encode and send response
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// AuthServerMetadataHandler handles requests for Authorization Server Metadata
// This provides information about GitHub's OAuth endpoints
type AuthServerMetadataHandler struct {
	config *Config
}

// NewAuthServerMetadataHandler creates a new handler for auth server metadata
func NewAuthServerMetadataHandler(config *Config) *AuthServerMetadataHandler {
	return &AuthServerMetadataHandler{
		config: config,
	}
}

// ServeHTTP implements http.Handler
func (h *AuthServerMetadataHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only allow GET requests
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Build the metadata response for GitHub as the authorization server
	metadata := AuthServerMetadata{
		Issuer:                h.config.ServerURL,
		AuthorizationEndpoint: h.config.ServerURL + "/oauth/authorize",
		TokenEndpoint:         h.config.ServerURL + "/oauth/token",
		// DCR is deprecated in MCP spec - clients should be pre-registered
		// RegistrationEndpoint:  h.config.GetRegistrationEndpointURL(),
		ScopesSupported:       h.config.ScopesSupported,
		ResponseTypesSupported: []string{
			"code", // Authorization code flow
		},
		GrantTypesSupported: []string{
			"authorization_code",
			"refresh_token",
		},
		TokenEndpointAuthMethodsSupported: []string{
			"client_secret_post",
			"client_secret_basic",
			"none", // Support public clients (like VS Code)
		},
		CodeChallengeMethodsSupported: []string{
			"S256", // PKCE with SHA-256
		},
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour

	// Encode and send response
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
