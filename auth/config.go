package auth

// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// Config holds the OAuth configuration for the MCP server
type Config struct {
	// ServerURL is the canonical URL of the MCP server (e.g., https://your-server.com or http://localhost:8080)
	ServerURL string

	// GitHub OAuth App credentials
	GitHubClientID     string
	GitHubClientSecret string

	// AllowedRedirectURIs is the list of valid redirect URIs for OAuth clients
	// Must include VS Code redirect URIs: http://127.0.0.1:33418 and https://vscode.dev/redirect
	AllowedRedirectURIs []string

	// ScopesSupported lists the scopes supported by this MCP server
	ScopesSupported []string

	// TokenExpiryDuration is how long access tokens remain valid
	TokenExpiryDuration time.Duration

	// EnforceHTTPS requires HTTPS for all OAuth operations (except localhost)
	EnforceHTTPS bool

	// OAuthEnabled controls whether OAuth authentication is enabled
	// If false, the server runs without authentication (for local development)
	OAuthEnabled bool

	// EnableDCR enables Dynamic Client Registration endpoint
	EnableDCR bool

	// AllowPublicClients allows registration of public clients (without client_secret)
	AllowPublicClients bool

	// GitHub API configuration
	GitHubAPIURL string

	// Authorization server endpoints (GitHub)
	GitHubAuthURL  string
	GitHubTokenURL string
}

// DefaultConfig returns a Config with default values
func DefaultConfig() *Config {
	return &Config{
		ServerURL: "http://localhost:8080",
		AllowedRedirectURIs: []string{
			"http://127.0.0.1:33418",
			"https://vscode.dev/redirect",
		},
		ScopesSupported: []string{
			"mcp:tools",
			"mcp:resources",
			"read:user",
		},
		TokenExpiryDuration: 1 * time.Hour,
		EnforceHTTPS:        false, // Default to false for development
		OAuthEnabled:        false, // Default to false for local development
		EnableDCR:           true,
		AllowPublicClients:  true,
		GitHubAPIURL:        "https://api.github.com",
		GitHubAuthURL:       "https://github.com/login/oauth/authorize",
		GitHubTokenURL:      "https://github.com/login/oauth/access_token",
	}
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() (*Config, error) {
	cfg := DefaultConfig()

	// Required: Server URL
	if serverURL := os.Getenv("MCP_SERVER_URL"); serverURL != "" {
		// Validate URL format
		parsedURL, err := url.Parse(serverURL)
		if err != nil {
			return nil, fmt.Errorf("invalid MCP_SERVER_URL: %w", err)
		}
		// Remove trailing slash for consistency
		cfg.ServerURL = strings.TrimSuffix(parsedURL.String(), "/")
	} else if host := os.Getenv("HOST"); host != "" && os.Getenv("PORT") != "" {
		port := os.Getenv("PORT")
		scheme := "http"
		if os.Getenv("USE_HTTPS") == "true" {
			scheme = "https"
		}
		cfg.ServerURL = fmt.Sprintf("%s://%s:%s", scheme, host, port)
	}

	// Required for OAuth: GitHub OAuth App credentials
	// First check for direct environment variables (local development)
	cfg.GitHubClientID = os.Getenv("GITHUB_CLIENT_ID")
	cfg.GitHubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")

	// If not found, check for AWS Secrets Manager secret name (production)
	if cfg.GitHubClientID == "" || cfg.GitHubClientSecret == "" {
		if secretName := os.Getenv("GITHUB_OAUTH_SECRET_NAME"); secretName != "" {
			// Load from AWS Secrets Manager
			if err := loadGitHubCredsFromSecretsManager(cfg, secretName); err != nil {
				return nil, fmt.Errorf("failed to load GitHub credentials from Secrets Manager: %w", err)
			}
		}
	}

	// Optional: Additional redirect URIs
	if redirectURIs := os.Getenv("OAUTH_REDIRECT_URIS"); redirectURIs != "" {
		uris := strings.Split(redirectURIs, ",")
		for _, uri := range uris {
			trimmed := strings.TrimSpace(uri)
			if trimmed != "" {
				// Validate redirect URI
				if _, err := url.Parse(trimmed); err != nil {
					return nil, fmt.Errorf("invalid redirect URI %s: %w", trimmed, err)
				}
				cfg.AllowedRedirectURIs = append(cfg.AllowedRedirectURIs, trimmed)
			}
		}
	}

	// Optional: Custom scopes
	if scopes := os.Getenv("OAUTH_SCOPES_SUPPORTED"); scopes != "" {
		cfg.ScopesSupported = strings.Split(scopes, ",")
		for i, scope := range cfg.ScopesSupported {
			cfg.ScopesSupported[i] = strings.TrimSpace(scope)
		}
	}

	// Optional: Token expiry
	if expiryStr := os.Getenv("TOKEN_EXPIRY_SECONDS"); expiryStr != "" {
		expiry, err := strconv.Atoi(expiryStr)
		if err != nil {
			return nil, fmt.Errorf("invalid TOKEN_EXPIRY_SECONDS: %w", err)
		}
		cfg.TokenExpiryDuration = time.Duration(expiry) * time.Second
	}

	// Optional: HTTPS enforcement
	if enforceHTTPS := os.Getenv("ENFORCE_HTTPS"); enforceHTTPS != "" {
		cfg.EnforceHTTPS = enforceHTTPS == "true" || enforceHTTPS == "1"
	}

	// Optional: OAuth enablement (defaults to false for local development)
	if oauthEnabled := os.Getenv("OAUTH_ENABLED"); oauthEnabled != "" {
		cfg.OAuthEnabled = oauthEnabled == "true" || oauthEnabled == "1"
	}

	// Optional: DCR enablement
	if enableDCR := os.Getenv("ENABLE_DCR"); enableDCR != "" {
		cfg.EnableDCR = enableDCR == "true" || enableDCR == "1"
	}

	// Optional: Public clients
	if allowPublic := os.Getenv("ALLOW_PUBLIC_CLIENTS"); allowPublic != "" {
		cfg.AllowPublicClients = allowPublic == "true" || allowPublic == "1"
	}

	// Optional: Custom GitHub URLs (for testing or GitHub Enterprise)
	if apiURL := os.Getenv("GITHUB_API_URL"); apiURL != "" {
		cfg.GitHubAPIURL = strings.TrimSuffix(apiURL, "/")
	}
	if authURL := os.Getenv("GITHUB_AUTH_URL"); authURL != "" {
		cfg.GitHubAuthURL = authURL
	}
	if tokenURL := os.Getenv("GITHUB_TOKEN_URL"); tokenURL != "" {
		cfg.GitHubTokenURL = tokenURL
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate server URL
	if c.ServerURL == "" {
		return fmt.Errorf("server URL is required")
	}
	parsedURL, err := url.Parse(c.ServerURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("server URL must use http or https scheme")
	}

	// Check HTTPS enforcement
	if c.EnforceHTTPS && parsedURL.Scheme == "http" && !isLocalhost(parsedURL.Host) {
		return fmt.Errorf("HTTPS enforcement enabled but server URL uses HTTP for non-localhost")
	}

	// Validate GitHub credentials if OAuth is enabled
	if c.OAuthEnabled {
		if c.GitHubClientID == "" {
			return fmt.Errorf("GitHub client ID is required when OAuth is enabled")
		}
		if c.GitHubClientSecret == "" && !c.AllowPublicClients {
			return fmt.Errorf("GitHub client secret is required when public clients are not allowed")
		}
	}

	// Validate redirect URIs
	if len(c.AllowedRedirectURIs) == 0 {
		return fmt.Errorf("at least one redirect URI must be configured")
	}
	for _, uri := range c.AllowedRedirectURIs {
		if _, err := url.Parse(uri); err != nil {
			return fmt.Errorf("invalid redirect URI %s: %w", uri, err)
		}
	}

	// Validate scopes
	if len(c.ScopesSupported) == 0 {
		return fmt.Errorf("at least one scope must be supported")
	}

	// Validate token expiry
	if c.TokenExpiryDuration <= 0 {
		return fmt.Errorf("token expiry duration must be positive")
	}

	return nil
}

// GetResourceMetadataURL returns the URL for the protected resource metadata endpoint
func (c *Config) GetResourceMetadataURL() string {
	return c.ServerURL + "/.well-known/oauth-protected-resource"
}

// GetRegistrationEndpointURL returns the URL for the dynamic client registration endpoint
func (c *Config) GetRegistrationEndpointURL() string {
	// DCR is not compatible with external authorization servers like GitHub
	// Clients must use the GitHub OAuth App credentials provided by the server operator
	if !c.EnableDCR || c.OAuthEnabled {
		return ""
	}
	return c.ServerURL + "/register"
}

// IsRedirectURIAllowed checks if a redirect URI is in the allowed list
func (c *Config) IsRedirectURIAllowed(uri string) bool {
	for _, allowed := range c.AllowedRedirectURIs {
		if uri == allowed {
			return true
		}
	}
	return false
}

// IsScopeSupported checks if a scope is supported
func (c *Config) IsScopeSupported(scope string) bool {
	for _, supported := range c.ScopesSupported {
		if scope == supported {
			return true
		}
	}
	return false
}

// isLocalhost checks if a host is localhost or 127.0.0.1
func isLocalhost(host string) bool {
	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

// loadGitHubCredsFromSecretsManager loads GitHub OAuth credentials from AWS Secrets Manager
func loadGitHubCredsFromSecretsManager(cfg *Config, secretName string) error {
	ctx := context.Background()

	// Load AWS SDK configuration
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("unable to load AWS SDK config: %w", err)
	}

	// Create Secrets Manager client
	client := secretsmanager.NewFromConfig(awsCfg)

	// Retrieve the secret
	result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretName,
	})
	if err != nil {
		return fmt.Errorf("failed to retrieve secret: %w", err)
	}

	// Parse the secret JSON
	var secrets struct {
		GitHubClientID     string `json:"GITHUB_CLIENT_ID"`
		GitHubClientSecret string `json:"GITHUB_CLIENT_SECRET"`
	}

	if err := json.Unmarshal([]byte(*result.SecretString), &secrets); err != nil {
		return fmt.Errorf("failed to parse secret JSON: %w", err)
	}

	// Set the credentials
	cfg.GitHubClientID = secrets.GitHubClientID
	cfg.GitHubClientSecret = secrets.GitHubClientSecret

	return nil
}
