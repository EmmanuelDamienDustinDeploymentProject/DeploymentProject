package auth

// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

import (
	"time"
)

// ProtectedResourceMetadata represents OAuth 2.0 Protected Resource Metadata per RFC 9728
// This metadata is served at /.well-known/oauth-protected-resource
type ProtectedResourceMetadata struct {
	// Resource is the canonical URI of the MCP server (RFC 8707)
	Resource string `json:"resource"`

	// AuthorizationServers lists the authorization servers that can issue tokens for this resource
	AuthorizationServers []string `json:"authorization_servers"`

	// ScopesSupported lists the scopes that this resource server supports
	ScopesSupported []string `json:"scopes_supported,omitempty"`

	// BearerMethodsSupported indicates supported bearer token methods (default: ["header"])
	BearerMethodsSupported []string `json:"bearer_methods_supported,omitempty"`

	// ResourceDocumentation provides a URL for resource documentation
	ResourceDocumentation string `json:"resource_documentation,omitempty"`
}

// AuthServerMetadata represents OAuth 2.0 Authorization Server Metadata per RFC 8414
type AuthServerMetadata struct {
	// Issuer is the authorization server's identifier
	Issuer string `json:"issuer"`

	// AuthorizationEndpoint is the URL of the authorization endpoint
	AuthorizationEndpoint string `json:"authorization_endpoint"`

	// TokenEndpoint is the URL of the token endpoint
	TokenEndpoint string `json:"token_endpoint"`

	// RegistrationEndpoint is the URL of the dynamic client registration endpoint (RFC 7591)
	RegistrationEndpoint string `json:"registration_endpoint,omitempty"`

	// ScopesSupported lists the supported OAuth scopes
	ScopesSupported []string `json:"scopes_supported,omitempty"`

	// ResponseTypesSupported lists the supported response types
	ResponseTypesSupported []string `json:"response_types_supported,omitempty"`

	// GrantTypesSupported lists the supported grant types
	GrantTypesSupported []string `json:"grant_types_supported,omitempty"`

	// TokenEndpointAuthMethodsSupported lists supported client authentication methods
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`

	// CodeChallengeMethodsSupported lists supported PKCE challenge methods
	CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported,omitempty"`
}

// ClientRegistrationRequest represents a Dynamic Client Registration request per RFC 7591
type ClientRegistrationRequest struct {
	// RedirectURIs is the array of redirection URI strings for use in redirect-based flows
	RedirectURIs []string `json:"redirect_uris,omitempty"`

	// TokenEndpointAuthMethod indicates the requested authentication method for the token endpoint
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method,omitempty"`

	// GrantTypes is the array of OAuth 2.0 grant type strings that the client can use
	GrantTypes []string `json:"grant_types,omitempty"`

	// ResponseTypes is the array of OAuth 2.0 response type strings that the client can use
	ResponseTypes []string `json:"response_types,omitempty"`

	// ClientName is the human-readable name of the client
	ClientName string `json:"client_name,omitempty"`

	// ClientURI is the URL string of a web page providing information about the client
	ClientURI string `json:"client_uri,omitempty"`

	// LogoURI is the URL string that references a logo for the client
	LogoURI string `json:"logo_uri,omitempty"`

	// Scope is a space-separated list of scope values
	Scope string `json:"scope,omitempty"`

	// Contacts is an array of strings representing ways to contact people responsible for this client
	Contacts []string `json:"contacts,omitempty"`

	// JWKSURI is the URL string referencing the client's JSON Web Key (JWK) Set
	JWKSURI string `json:"jwks_uri,omitempty"`

	// SoftwareID is a unique identifier for the client software
	SoftwareID string `json:"software_id,omitempty"`

	// SoftwareVersion is a version identifier for the client software
	SoftwareVersion string `json:"software_version,omitempty"`
}

// ClientRegistrationResponse represents the response to a successful client registration
type ClientRegistrationResponse struct {
	// ClientID is the unique client identifier assigned by the authorization server
	ClientID string `json:"client_id"`

	// ClientSecret is the client secret (optional, for confidential clients)
	ClientSecret string `json:"client_secret,omitempty"`

	// ClientIDIssuedAt is the time at which the client identifier was issued
	ClientIDIssuedAt int64 `json:"client_id_issued_at,omitempty"`

	// ClientSecretExpiresAt is the time at which the client secret will expire (0 if it will not expire)
	ClientSecretExpiresAt int64 `json:"client_secret_expires_at,omitempty"`

	// All registered metadata is returned
	RedirectURIs            []string `json:"redirect_uris,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	ClientName              string   `json:"client_name,omitempty"`
	ClientURI               string   `json:"client_uri,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
	Scope                   string   `json:"scope,omitempty"`
	Contacts                []string `json:"contacts,omitempty"`
	JWKSURI                 string   `json:"jwks_uri,omitempty"`
	SoftwareID              string   `json:"software_id,omitempty"`
	SoftwareVersion         string   `json:"software_version,omitempty"`
}

// ClientRegistrationError represents an error response from the registration endpoint
type ClientRegistrationError struct {
	// Error is the error code
	Error string `json:"error"`

	// ErrorDescription is a human-readable description of the error
	ErrorDescription string `json:"error_description,omitempty"`
}

// Standard error codes for client registration per RFC 7591
const (
	ErrorInvalidRedirectURI          = "invalid_redirect_uri"
	ErrorInvalidClientMetadata       = "invalid_client_metadata"
	ErrorInvalidSoftwareStatement    = "invalid_software_statement"
	ErrorUnapprovedSoftwareStatement = "unapproved_software_statement"
)

// OAuthClient represents a registered OAuth client
type OAuthClient struct {
	// ClientID is the unique client identifier
	ClientID string `json:"client_id"`

	// ClientSecret is the client secret (hashed for storage)
	ClientSecret string `json:"client_secret,omitempty"`

	// Metadata contains the client's registered metadata
	Metadata ClientRegistrationRequest `json:"metadata"`

	// CreatedAt is the timestamp when the client was registered
	CreatedAt time.Time `json:"created_at"`

	// ExpiresAt is the timestamp when the client registration expires (optional)
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// TokenValidationResult represents the result of validating an OAuth access token
type TokenValidationResult struct {
	// Valid indicates whether the token is valid
	Valid bool

	// ClientID is the client identifier associated with the token
	ClientID string

	// Scopes is the list of scopes granted to the token
	Scopes []string

	// Subject is the user identifier (GitHub username)
	Subject string

	// ExpiresAt is when the token expires
	ExpiresAt time.Time

	// GitHubUser contains the GitHub user information
	GitHubUser *GitHubUserInfo

	// Error contains validation error details if Valid is false
	Error error
}

// GitHubUserInfo represents GitHub user information from the API
type GitHubUserInfo struct {
	Login     string `json:"login"`
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

// PKCEChallenge represents a PKCE code challenge
type PKCEChallenge struct {
	// CodeVerifier is the high-entropy cryptographic random string (43-128 characters)
	CodeVerifier string

	// CodeChallenge is the transformed version of the code verifier
	CodeChallenge string

	// CodeChallengeMethod is the transformation method (S256 or plain)
	CodeChallengeMethod string
}

// OAuthError represents a standard OAuth error response
type OAuthError struct {
	// Error is the error code
	Error string `json:"error"`

	// ErrorDescription is a human-readable description
	ErrorDescription string `json:"error_description,omitempty"`

	// ErrorURI is a URI with information about the error
	ErrorURI string `json:"error_uri,omitempty"`
}

// Standard OAuth error codes
const (
	ErrorInvalidRequest = "invalid_request"
	ErrorServerError    = "server_error"
)
