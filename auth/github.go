// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
)

// GitHubTokenVerifier implements the MCP SDK's auth.TokenVerifier interface
// It validates access tokens issued by our OAuth server
type GitHubTokenVerifier struct {
	config       *Config
	httpClient   *http.Client
	cache        TokenCache
	tokenStorage TokenStorage
}

// NewGitHubTokenVerifier creates a new GitHub token verifier
func NewGitHubTokenVerifier(config *Config, cache TokenCache, tokenStorage TokenStorage) *GitHubTokenVerifier {
	return &GitHubTokenVerifier{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:        cache,
		tokenStorage: tokenStorage,
	}
}

// Verify implements auth.TokenVerifier
// This is called by the MCP SDK's RequireBearerToken middleware
func (v *GitHubTokenVerifier) Verify(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
	// Look up token in our storage
	tokenInfo, err := v.tokenStorage.GetAccessToken(token)
	if err != nil {
		return nil, fmt.Errorf("%w: token not found or expired", auth.ErrInvalidToken)
	}

	// Check cache for GitHub token validation
	cacheKey := "github:" + tokenInfo.GitHubAccessToken
	if v.cache != nil {
		if cached, found := v.cache.Get(cacheKey); found {
			if cached.Valid {
				// Convert our TokenValidationResult to SDK's TokenInfo
				return &auth.TokenInfo{
					Scopes:     strings.Split(tokenInfo.Scope, " "),
					Expiration: tokenInfo.ExpiresAt,
					Extra: map[string]any{
						"github_user": cached.GitHubUser,
						"subject":     cached.Subject,
						"client_id":   tokenInfo.ClientID,
						"resource":    tokenInfo.Resource,
					},
				}, nil
			}
			// Cached but invalid
			return nil, fmt.Errorf("%w: %v", auth.ErrInvalidToken, cached.Error)
		}
	}

	// Validate GitHub token with GitHub API
	result := v.validateWithGitHub(ctx, tokenInfo.GitHubAccessToken)

	// Cache the GitHub validation result
	if v.cache != nil {
		_ = v.cache.Set(cacheKey, result, v.config.TokenExpiryDuration)
	}

	if !result.Valid {
		return nil, fmt.Errorf("%w: %v", auth.ErrInvalidToken, result.Error)
	}

	// Convert to SDK's TokenInfo
	return &auth.TokenInfo{
		Scopes:     strings.Split(tokenInfo.Scope, " "),
		Expiration: tokenInfo.ExpiresAt,
		Extra: map[string]any{
			"github_user": result.GitHubUser,
			"subject":     result.Subject,
			"client_id":   tokenInfo.ClientID,
			"resource":    tokenInfo.Resource,
		},
	}, nil
}

// validateWithGitHub validates the token by calling GitHub's API
func (v *GitHubTokenVerifier) validateWithGitHub(ctx context.Context, token string) *TokenValidationResult {
	// Call GitHub API to verify token and get user info
	req, err := http.NewRequestWithContext(ctx, "GET", v.config.GitHubAPIURL+"/user", nil)
	if err != nil {
		return &TokenValidationResult{
			Valid: false,
			Error: fmt.Errorf("failed to create request: %w", err),
		}
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return &TokenValidationResult{
			Valid: false,
			Error: fmt.Errorf("failed to call GitHub API: %w", err),
		}
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &TokenValidationResult{
			Valid: false,
			Error: fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body)),
		}
	}

	var user GitHubUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return &TokenValidationResult{
			Valid: false,
			Error: fmt.Errorf("failed to decode GitHub response: %w", err),
		}
	}

	// Get the scopes from the X-OAuth-Scopes header
	scopes := parseGitHubScopes(resp.Header.Get("X-OAuth-Scopes"))

	// Validate that required MCP scopes are present
	// For GitHub, we map GitHub scopes to MCP scopes
	mcpScopes := mapGitHubScopesToMCP(scopes)

	// Set expiration based on configuration
	expiresAt := time.Now().Add(v.config.TokenExpiryDuration)

	return &TokenValidationResult{
		Valid:      true,
		Scopes:     mcpScopes,
		Subject:    user.Login,
		ExpiresAt:  expiresAt,
		GitHubUser: &user,
		Error:      nil,
	}
}

// parseGitHubScopes parses the X-OAuth-Scopes header from GitHub
func parseGitHubScopes(scopeHeader string) []string {
	if scopeHeader == "" {
		return []string{}
	}

	scopes := strings.Split(scopeHeader, ",")
	result := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		trimmed := strings.TrimSpace(scope)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// mapGitHubScopesToMCP maps GitHub OAuth scopes to MCP scopes
// This provides a flexible mapping between GitHub permissions and MCP tool access
func mapGitHubScopesToMCP(githubScopes []string) []string {
	mcpScopes := make([]string, 0)

	// Always add read:user if the user authenticated
	mcpScopes = append(mcpScopes, "read:user")

	// Map GitHub scopes to MCP scopes
	for _, scope := range githubScopes {
		switch scope {
		case "repo", "public_repo", "read:repo_hook":
			// Repository access grants mcp:resources
			if !contains(mcpScopes, "mcp:resources") {
				mcpScopes = append(mcpScopes, "mcp:resources")
			}
		case "workflow", "write:repo_hook", "admin:repo_hook":
			// Write access grants mcp:tools
			if !contains(mcpScopes, "mcp:tools") {
				mcpScopes = append(mcpScopes, "mcp:tools")
			}
		case "read:user", "user", "user:email":
			// User scopes are already included
			continue
		default:
			// Include other GitHub scopes as-is for extensibility
			if !contains(mcpScopes, scope) {
				mcpScopes = append(mcpScopes, scope)
			}
		}
	}

	// If no specific mappings were found, provide basic access
	if len(mcpScopes) == 1 { // Only has read:user
		mcpScopes = append(mcpScopes, "mcp:tools", "mcp:resources")
	}

	return mcpScopes
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
