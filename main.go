// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"EmmanuelDamienDustinDeploymentProject/DeploymentProject/auth"
	"EmmanuelDamienDustinDeploymentProject/DeploymentProject/tools"
)

func main() {
	host := os.Getenv("HOST")
	port := os.Getenv("PORT")
	if host == "" {
		host = "0.0.0.0"
	}
	if port == "" {
		port = "8080"
	}
	runServer(fmt.Sprintf("%s:%s", host, port))
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// Allow CORS for localhost:6277 (MCP Inspector) and localhost:6274
		if origin == "http://localhost:6277" || origin == "http://localhost:6274" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, mcp-protocol-version")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "3600")
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func runServer(url string) {
	// Load OAuth configuration
	config, err := auth.LoadConfigFromEnv()
	if err != nil {
		log.Printf("Warning: Failed to load OAuth config: %v. OAuth will be disabled.", err)
		runServerWithoutAuth(url)
		return
	}

	if err := config.Validate(); err != nil {
		log.Printf("Warning: Invalid OAuth config: %v. OAuth will be disabled.", err)
		runServerWithoutAuth(url)
		return
	}

	// Initialize OAuth components
	clientStorage := auth.NewInMemoryClientStorage()
	tokenCache := auth.NewInMemoryTokenCache()
	githubVerifier := auth.NewGitHubTokenVerifier(config, tokenCache)
	middleware := auth.NewMiddleware(config, githubVerifier)

	// Create an MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "time-server",
		Version: "1.0.0",
	}, nil)

	tools.RegisterAll(server)

	// Create the streamable HTTP handler with session timeout
	// Sessions are needed for GET requests (SSE streaming)
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		SessionTimeout: 30 * time.Minute, // Automatically close idle sessions after 30 minutes
	})

	// Wrap MCP handler with OAuth authentication, but allow GET requests with session ID
	// GET requests are used for SSE streaming and may not include Authorization header
	authenticatedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow GET requests that have a session ID (for SSE streaming)
		if r.Method == http.MethodGet && r.Header.Get("Mcp-Session-Id") != "" {
			handler.ServeHTTP(w, r)
			return
		}
		// All other requests require OAuth authentication
		middleware.RequireAuth([]string{"mcp:tools"})(handler).ServeHTTP(w, r)
	})

	// Set up routes
	mux := http.NewServeMux()

	// Public endpoints (no authentication required)
	mux.HandleFunc("/health", healthCheckHandler)
	mux.Handle("/.well-known/oauth-protected-resource",
		auth.NewProtectedResourceMetadataHandler(config))
	mux.Handle("/.well-known/oauth-authorization-server",
		auth.NewAuthServerMetadataHandler(config))

	// DCR endpoint (if enabled)
	if config.EnableDCR {
		mux.Handle("/register", auth.NewRegistrationHandler(config, clientStorage))
		log.Printf("Dynamic Client Registration enabled at /register")
	}

	// OAuth proxy endpoints to avoid CORS issues
	mux.Handle("/oauth/authorize", auth.NewAuthorizeProxyHandler(config))
	mux.Handle("/oauth/token", auth.NewTokenProxyHandler(config))
	mux.Handle("/oauth/callback", auth.NewCallbackHandler(config))

	// Protected MCP endpoint
	mux.Handle("/", authenticatedHandler)

	handlerWithLogging := loggingHandler(corsMiddleware(mux))

	log.Printf("MCP server listening on %s", url)
	log.Printf("OAuth 2.1 authentication enabled with GitHub")
	log.Printf("Protected Resource Metadata: /.well-known/oauth-protected-resource")
	log.Printf("Authorization Server Metadata: /.well-known/oauth-authorization-server")
	log.Printf("Available tool: Get City Time (cities: nyc, sf, boston)")
	log.Printf("Available tool: Get Fortune")
	log.Printf("Available tool: APR Calculator")
	log.Printf("Health check available at /health")

	// Start the HTTP server with logging handler
	if err := http.ListenAndServe(url, handlerWithLogging); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func runServerWithoutAuth(url string) {
	// Create an MCP server without authentication
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "time-server",
		Version: "1.0.0",
	}, nil)

	tools.RegisterAll(server)

	// Create the streamable HTTP handler
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, nil)

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.HandleFunc("/health", healthCheckHandler)

	handlerWithLogging := loggingHandler(corsMiddleware(mux))

	log.Printf("MCP server listening on %s", url)
	log.Printf("Health check available at /health")

	// Start the HTTP server with logging handler
	if err := http.ListenAndServe(url, handlerWithLogging); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
