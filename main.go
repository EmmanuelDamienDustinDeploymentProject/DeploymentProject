// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"EmmanuelDamienDustinDeploymentProject/DeploymentProject/tools"
)

func main() {
	// Initialize OAuth configuration
	InitializeOAuth()

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

func runServer(url string) {
	// Create an MCP server.
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "time-server",
		Version: "1.0.0",
	}, &mcp.ServerOptions{
		HasTools:  true,
		KeepAlive: 10,
	})

	tools.RegisterAll(server)

	// Create the streamable HTTP handler.
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, nil)

	mux := http.NewServeMux()

	// Register specific routes first (before the catch-all "/" route)
	mux.HandleFunc("/health", healthCheckHandler)
	mux.HandleFunc("/oauth/login", oauthLoginHandler)
	mux.HandleFunc("/oauth/callback", oauthCallbackHandler)
	mux.HandleFunc("/oauth/token", oauthTokenHandler)
	mux.HandleFunc("/dcr", dcrHandler)

	// OAuth discovery endpoints (RFC 8414 and RFC 9728) - must be public
	mux.HandleFunc("/.well-known/oauth-protected-resource", oauthMetadataHandler)
	mux.HandleFunc("/.well-known/oauth-authorization-server", authServerMetadataHandler)

	// MCP endpoint with authentication middleware
	mux.Handle("/", bearerTokenMiddleware(handler))

	// Wrap the mux with CORS middleware and then logging middleware
	handlerWithCORS := corsMiddleware(mux)
	handlerWithLogging := loggingHandler(handlerWithCORS)

	log.Printf("MCP server listening on %s", url)
	log.Printf("Available tool: cityTime (cities: nyc, sf, boston)")
	log.Printf("Health check available at /health")

	// Start the HTTP server with logging handler.
	if err := http.ListenAndServe(url, handlerWithLogging); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow all origins
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// Allow methods
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		// Allow headers
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Pass down the request to the next handler
		next.ServeHTTP(w, r)
	})
}
