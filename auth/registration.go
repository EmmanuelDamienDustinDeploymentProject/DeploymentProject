// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// RegistrationHandler handles Dynamic Client Registration requests per RFC 7591
type RegistrationHandler struct {
	config  *Config
	storage ClientStorage
}

// NewRegistrationHandler creates a new DCR handler
func NewRegistrationHandler(config *Config, storage ClientStorage) *RegistrationHandler {
	return &RegistrationHandler{
		config:  config,
		storage: storage,
	}
}

// ServeHTTP implements http.Handler for the /register endpoint
func (h *RegistrationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		h.sendError(w, ErrorInvalidRequest, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if DCR is enabled
	if !h.config.EnableDCR {
		h.sendError(w, ErrorInvalidRequest, "Dynamic client registration is not enabled", http.StatusForbidden)
		return
	}

	// Parse request body
	var req ClientRegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, ErrorInvalidRequest, "Invalid JSON in request body", http.StatusBadRequest)
		return
	}

	// Validate the registration request
	if err := h.validateRequest(&req); err != nil {
		h.sendError(w, ErrorInvalidClientMetadata, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate client credentials
	clientID, err := GenerateClientID()
	if err != nil {
		h.sendError(w, ErrorServerError, "Failed to generate client ID", http.StatusInternalServerError)
		return
	}

	var clientSecret string
	var hashedSecret string

	// Generate client secret for confidential clients
	if req.TokenEndpointAuthMethod != "none" {
		clientSecret, err = GenerateClientSecret()
		if err != nil {
			h.sendError(w, ErrorServerError, "Failed to generate client secret", http.StatusInternalServerError)
			return
		}
		hashedSecret = hashSecret(clientSecret)
	}

	// Apply defaults
	h.applyDefaults(&req)

	// Create the OAuth client
	now := time.Now()
	client := &OAuthClient{
		ClientID:     clientID,
		ClientSecret: hashedSecret,
		Metadata:     req,
		CreatedAt:    now,
	}

	// Store the client
	if err := h.storage.StoreClient(client); err != nil {
		h.sendError(w, ErrorServerError, "Failed to store client registration", http.StatusInternalServerError)
		return
	}

	// Build response
	response := ClientRegistrationResponse{
		ClientID:                clientID,
		ClientSecret:            clientSecret, // Return plaintext secret only once
		ClientIDIssuedAt:        now.Unix(),
		ClientSecretExpiresAt:   0, // Secrets don't expire by default
		RedirectURIs:            req.RedirectURIs,
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		GrantTypes:              req.GrantTypes,
		ResponseTypes:           req.ResponseTypes,
		ClientName:              req.ClientName,
		ClientURI:               req.ClientURI,
		LogoURI:                 req.LogoURI,
		Scope:                   req.Scope,
		Contacts:                req.Contacts,
		JWKSURI:                 req.JWKSURI,
		SoftwareID:              req.SoftwareID,
		SoftwareVersion:         req.SoftwareVersion,
	}

	// Set headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusCreated)

	// Send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Too late to change status code, but log the error
		fmt.Printf("Failed to encode response: %v\n", err)
	}
}

// validateRequest validates the client registration request
func (h *RegistrationHandler) validateRequest(req *ClientRegistrationRequest) error {
	// Validate redirect URIs
	if len(req.RedirectURIs) == 0 {
		return fmt.Errorf("at least one redirect_uri is required")
	}

	for _, uri := range req.RedirectURIs {
		if uri == "" {
			return fmt.Errorf("redirect_uri cannot be empty")
		}

		// For VS Code compatibility, ensure the required redirect URIs are present
		// But allow additional ones
		// Validate URI format (basic check)
		if len(uri) > 2048 {
			return fmt.Errorf("redirect_uri too long: %s", uri)
		}
	}

	// Validate grant types
	if len(req.GrantTypes) > 0 {
		validGrantTypes := map[string]bool{
			"authorization_code": true,
			"implicit":           true,
			"password":           true,
			"client_credentials": true,
			"refresh_token":      true,
		}
		for _, gt := range req.GrantTypes {
			if !validGrantTypes[gt] {
				return fmt.Errorf("invalid grant_type: %s", gt)
			}
		}
	}

	// Validate response types
	if len(req.ResponseTypes) > 0 {
		validResponseTypes := map[string]bool{
			"code":  true,
			"token": true,
		}
		for _, rt := range req.ResponseTypes {
			if !validResponseTypes[rt] {
				return fmt.Errorf("invalid response_type: %s", rt)
			}
		}
	}

	// Validate token endpoint auth method
	if req.TokenEndpointAuthMethod != "" {
		validMethods := map[string]bool{
			"none":                true,
			"client_secret_post":  true,
			"client_secret_basic": true,
		}
		if !validMethods[req.TokenEndpointAuthMethod] {
			return fmt.Errorf("invalid token_endpoint_auth_method: %s", req.TokenEndpointAuthMethod)
		}

		// Check if public clients are allowed
		if req.TokenEndpointAuthMethod == "none" && !h.config.AllowPublicClients {
			return fmt.Errorf("public clients are not allowed")
		}
	}

	// Validate client name length
	if len(req.ClientName) > 256 {
		return fmt.Errorf("client_name too long (max 256 characters)")
	}

	return nil
}

// applyDefaults applies default values to the registration request
func (h *RegistrationHandler) applyDefaults(req *ClientRegistrationRequest) {
	// Default token endpoint auth method
	if req.TokenEndpointAuthMethod == "" {
		if h.config.AllowPublicClients {
			req.TokenEndpointAuthMethod = "none"
		} else {
			req.TokenEndpointAuthMethod = "client_secret_basic"
		}
	}

	// Default grant types
	if len(req.GrantTypes) == 0 {
		req.GrantTypes = []string{"authorization_code"}
	}

	// Default response types
	if len(req.ResponseTypes) == 0 {
		req.ResponseTypes = []string{"code"}
	}

	// Default scope
	if req.Scope == "" {
		req.Scope = "mcp:tools mcp:resources read:user"
	}
}

// sendError sends an error response
func (h *RegistrationHandler) sendError(w http.ResponseWriter, errorCode, description string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(statusCode)

	errorResp := ClientRegistrationError{
		Error:            errorCode,
		ErrorDescription: description,
	}

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}
