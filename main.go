// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"EmmanuelDamienDustinDeploymentProject/DeploymentProject/auth"
	"EmmanuelDamienDustinDeploymentProject/DeploymentProject/chat"
	"EmmanuelDamienDustinDeploymentProject/DeploymentProject/prompts"
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

	// Check if OAuth is enabled
	if !config.OAuthEnabled {
		log.Printf("OAuth is disabled (set OAUTH_ENABLED=true to enable)")
		runServerWithoutAuth(url)
		return
	}

	if err := config.Validate(); err != nil {
		log.Printf("Warning: Invalid OAuth config: %v. OAuth will be disabled.", err)
		runServerWithoutAuth(url)
		return
	}

	// Initialize OAuth components with default clients
	clientStorage := auth.NewInMemoryClientStorageWithDefaults()
	tokenStorage := auth.NewInMemoryTokenStorage()
	tokenCache := auth.NewInMemoryTokenCache()
	githubVerifier := auth.NewGitHubTokenVerifier(config, tokenCache, tokenStorage)
	middleware := auth.NewMiddleware(config, githubVerifier)
	
	log.Printf("Pre-registered OAuth client: vscode (client_id can be used in MCP config)")

	// Create authorization handler with state store
	authHandler := auth.NewAuthorizationHandler(config, clientStorage)

	// Create callback handler that shares the state store
	callbackHandler := auth.NewCallbackHandler(config, authHandler.GetStateStore(), tokenStorage)

	// Create token endpoint handler
	tokenHandler := auth.NewTokenEndpointHandler(config, clientStorage, tokenStorage)

	// Create chat server
	chatServer := chat.NewServer()
	log.Printf("Chat server initialized")

	// Create an MCP server
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "chat-relay-server",
		Version: "1.0.0",
	}, nil)

	tools.RegisterAll(server)
	prompts.RegisterAll(server)
	
	// Register chat tools
	chatSendTool := tools.NewSendChatMessage(chatServer)
	chatSendTool.Register(server)
	log.Printf("Registered tool: %s", chatSendTool.Name)
	
	chatHistoryTool := tools.NewGetChatHistory(chatServer)
	chatHistoryTool.Register(server)
	log.Printf("Registered tool: %s", chatHistoryTool.Name)
	
	chatUsersTool := tools.NewListActiveUsers(chatServer)
	chatUsersTool.Register(server)
	log.Printf("Registered tool: %s", chatUsersTool.Name)

	// Create the streamable HTTP handler with session timeout
	// Sessions are needed for GET requests (SSE streaming)
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		SessionTimeout: 30 * time.Minute, // Automatically close idle sessions after 30 minutes
	})

	// Wrap MCP handler with OAuth authentication and chat connection tracking
	authenticatedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.Header.Get("Mcp-Session-Id")
		
		// Allow GET requests that have a session ID (for SSE streaming)
		if r.Method == http.MethodGet && sessionID != "" {
			handler.ServeHTTP(w, r)
			return
		}
		
		// All other requests require OAuth authentication
		authMiddleware := middleware.RequireAuth([]string{"mcp:tools"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract GitHub username from token info and register chat connection
			if tokenInfo := r.Context().Value("tokenInfo"); tokenInfo != nil {
				if ti, ok := tokenInfo.(*auth.AccessTokenInfo); ok {
					if sessionID != "" {
						// Get GitHub username (use ClientID as fallback)
						githubUser := ti.ClientID
						
						// Try to get actual GitHub user from the token verifier
						if ti.GitHubAccessToken != "" {
							if result, err := githubVerifier.Verify(r.Context(), ti.GitHubAccessToken, r); err == nil {
								if ghUser, ok := result.Extra["github_user"].(*auth.GitHubUserInfo); ok {
									githubUser = ghUser.Login
								}
							}
						}
						
						// Register or update connection
						if _, exists := chatServer.GetConnection(sessionID); !exists {
							chatServer.RegisterConnection(sessionID, githubUser)
							log.Printf("Registered chat connection for user: %s (session: %s)", githubUser, sessionID)
						}
						
						// Add sessionID to context for tools to use
						ctx := context.WithValue(r.Context(), "sessionID", sessionID)
						r = r.WithContext(ctx)
					}
				}
			}
			
			handler.ServeHTTP(w, r)
		}))
		
		authMiddleware.ServeHTTP(w, r)
	})

	// Set up routes
	mux := http.NewServeMux()

	// Public endpoints (no authentication required)
	mux.HandleFunc("/health", healthCheckHandler)
	mux.Handle("/.well-known/oauth-protected-resource",
		auth.NewProtectedResourceMetadataHandler(config))
	mux.Handle("/.well-known/oauth-authorization-server",
		auth.NewAuthServerMetadataHandler(config))
	// Alias for OpenID Connect discovery (VS Code compatibility)
	mux.Handle("/.well-known/openid-configuration",
		auth.NewAuthServerMetadataHandler(config))

	// DCR endpoint (if enabled)
	if config.EnableDCR {
		mux.Handle("/register", auth.NewRegistrationHandler(config, clientStorage))
		log.Printf("Dynamic Client Registration enabled at /register")
	}

	// OAuth endpoints (proper OAuth 2.1 flow with DCR support)
	mux.Handle("/oauth/authorize", authHandler)
	mux.Handle("/oauth/token", tokenHandler)
	mux.Handle("/oauth/callback", callbackHandler)

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
	prompts.RegisterAll(server)

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
