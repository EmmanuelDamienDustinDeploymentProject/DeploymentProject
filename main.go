// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"EmmanuelDamienDustinDeploymentProject/DeploymentProject/tools"
)

// TODO: Move this to environment variables
var (
	host = flag.String("host", "0.0.0.0", "host to connect to/listen on")
	port = flag.Int("port", 8080, "port number to connect to/listen on")
)

func main() {
	runServer(fmt.Sprintf("%s:%d", *host, *port))
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
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
	mux.Handle("/", handler)
	mux.HandleFunc("/health", healthCheckHandler)

	handlerWithLogging := loggingHandler(mux)

	log.Printf("MCP server listening on %s", url)
	log.Printf("Health check available at /health")

	// Start the HTTP server with logging handler.
	if err := http.ListenAndServe(url, handlerWithLogging); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
