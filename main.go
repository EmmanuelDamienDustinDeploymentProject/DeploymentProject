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
	host := os.Getenv("HOST")
	port := os.Getenv("PORT")
	if host == "" {
		host = "0.0.0.0"
	}
	if port == "" {
		port = "8080"
	}

	// Initialize OAuth configuration
	InitOAuth()

	runServer(fmt.Sprintf("%s:%s", host, port))
}

func runServer(url string) {
	// Create an MCP server.
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "time-server",
		Version: "1.0.0",
	}, nil)

	tools.RegisterAll(server)

	// Create the streamable HTTP handler.
	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, nil)

	mux := http.NewServeMux()

	// this is the mcp endpoint (requires authentication)
	mux.Handle("/", handler)

	mux.HandleFunc("/health", healthCheckHandler)
	mux.HandleFunc("/oauth/login", oauthLoginHandler)
	mux.HandleFunc("/oauth/callback", oauthCallbackHandler)

	handlerWithLogging := loggingHandler(mux)

	log.Printf("MCP server listening on %s", url)
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
