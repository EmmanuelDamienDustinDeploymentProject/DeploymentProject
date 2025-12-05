// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/auth"
)

// Middleware provides OAuth middleware integration with the MCP server
type Middleware struct {
	config   *Config
	verifier *GitHubTokenVerifier
}

// NewMiddleware creates a new OAuth middleware
func NewMiddleware(config *Config, verifier *GitHubTokenVerifier) *Middleware {
	return &Middleware{
		config:   config,
		verifier: verifier,
	}
}

// RequireAuth returns HTTP middleware that requires OAuth authentication
// This wraps the MCP SDK's auth.RequireBearerToken with our GitHub token verifier
// Special handling: GET requests are allowed through without token validation to support SSE streaming
// The MCP handler will validate the session ID
func (m *Middleware) RequireAuth(scopes []string) func(http.Handler) http.Handler {
	// Create the MCP SDK middleware with our verifier
	opts := &auth.RequireBearerTokenOptions{
		ResourceMetadataURL: m.config.GetResourceMetadataURL(),
		Scopes:              scopes,
	}

	sdkMiddleware := auth.RequireBearerToken(
		func(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
			return m.verifier.Verify(ctx, token, req)
		},
		opts,
	)

	// Wrap the SDK middleware to allow GET requests and expose tokenInfo
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow GET requests to pass through for SSE streaming
			// The MCP handler will validate the session ID
			if r.Method == http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			// For all other requests (POST, etc.), apply OAuth authentication
			// Wrap the next handler to capture tokenInfo
			wrappedNext := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// The SDK middleware sets tokenInfo in context with key "tokenInfo"
				if ti := r.Context().Value("tokenInfo"); ti != nil {
					// Re-add it to ensure it's available
					ctx := context.WithValue(r.Context(), "tokenInfo", ti)
					r = r.WithContext(ctx)
				}
				next.ServeHTTP(w, r)
			})
			
			sdkMiddleware(wrappedNext).ServeHTTP(w, r)
		})
	}
}

// OptionalAuth returns HTTP middleware that allows but doesn't require authentication
// If a token is present, it will be validated. If not present, the request proceeds.
func (m *Middleware) OptionalAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if Authorization header is present
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				// No token, proceed without authentication
				next.ServeHTTP(w, r)
				return
			}

			// Token present, validate it
			tokenInfo, err := m.verifier.Verify(r.Context(), extractBearerToken(authHeader), r)
			if err != nil {
				// Invalid token, but we allow the request to proceed
				// The handler can check if TokenInfo is present in context
				next.ServeHTTP(w, r)
				return
			}

			// Add token info to context
			ctx := context.WithValue(r.Context(), tokenInfoKey{}, tokenInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractBearerToken extracts the token from a Bearer authorization header
func extractBearerToken(authHeader string) string {
	const prefix = "Bearer "
	if len(authHeader) > len(prefix) && authHeader[:len(prefix)] == prefix {
		return authHeader[len(prefix):]
	}
	return ""
}

// tokenInfoKey is the context key for TokenInfo
type tokenInfoKey struct{}
