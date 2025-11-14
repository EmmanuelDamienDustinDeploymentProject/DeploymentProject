// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TODO: Move this to environment variables
var (
	host = flag.String("host", "0.0.0.0", "host to connect to/listen on")
	port = flag.Int("port", 8080, "port number to connect to/listen on")
)

func main() {
	flag.Parse()

	// Initialize OAuth configuration
	InitOAuth()

	runServer(fmt.Sprintf("%s:%d", *host, *port))
}

// GetTimeParams defines the parameters for the cityTime tool.
type GetTimeParams struct {
	City string `json:"city" jsonschema:"City to get time for (nyc, sf, or boston)"`
}

// getTime implements the tool that returns the current time for a given city.
func getTime(_ context.Context, _ *mcp.CallToolRequest, params *GetTimeParams) (*mcp.CallToolResult, any, error) {
	// Define time zones for each city
	locations := map[string]string{
		"nyc":    "America/New_York",
		"sf":     "America/Los_Angeles",
		"boston": "America/New_York",
	}

	city := params.City
	if city == "" {
		city = "nyc" // Default to NYC
	}

	// Get the timezone.
	tzName, ok := locations[city]
	if !ok {
		return nil, nil, fmt.Errorf("unknown city: %s", city)
	}

	// Load the location.
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load timezone: %w", err)
	}

	// Get current time in that location.
	now := time.Now().In(loc)

	// Format the response.
	cityNames := map[string]string{
		"nyc":    "New York City",
		"sf":     "San Francisco",
		"boston": "Boston",
	}

	response := fmt.Sprintf("The current time in %s is %s",
		cityNames[city],
		now.Format(time.RFC3339))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: response},
		},
	}, nil, nil
}

func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func runServer(url string) {
	// Determine MCP handler options from environment (default: stateless true unless explicitly disabled)
	envStateless := strings.ToLower(os.Getenv("MCP_STATELESS"))
	useStateless := true
	if envStateless == "0" || envStateless == "false" {
		useStateless = false
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "time-server",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "cityTime",
		Description: "Get the current time in NYC, San Francisco, or Boston",
	}, getTime)

	mcpHandler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		Stateless:    useStateless,
		JSONResponse: useStateless, // if stateless, make curl output easier
	})

	mux := http.NewServeMux()

	// Root info page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		modeStr := "session"
		transportStr := " SSE stream"
		if useStateless {
			modeStr = "stateless"
			transportStr = " JSON response"
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "MCP Time Server\n\nMode: %s\nEndpoints:\n  /oauth/login (start GitHub OAuth)\n  /oauth/callback (OAuth redirect)\n  /mcp (MCP transport%s)\n  /time?city=nyc (REST time query, bearer required)\n  /whoami (REST auth test, bearer required)\n  /health (health check)\n\nTools:\n  cityTime: cities = nyc, sf, boston\n", modeStr, transportStr)
	})

	// OAuth endpoints (no authentication required)
	mux.HandleFunc("/oauth/login", oauthLoginHandler)
	mux.HandleFunc("/oauth/callback", oauthCallbackHandler)

	// MCP endpoint (requires authentication)
	mux.Handle("/mcp", bearerTokenMiddleware(mcpHandler))

	// Simple REST convenience endpoint for manual curl testing
	mux.Handle("/time", bearerTokenMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		city := r.URL.Query().Get("city")
		result, _, err := getTime(r.Context(), nil, &GetTimeParams{City: city})
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		if len(result.Content) == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("empty response"))
			return
		}
		if tc, ok := result.Content[0].(*mcp.TextContent); ok {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(tc.Text))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("unexpected content type"))
	})))

	// Authenticated whoami endpoint to return the GitHub username linked to the bearer token
	mux.Handle("/whoami", bearerTokenMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("missing bearer token"))
			return
		}
		token := strings.TrimPrefix(authHeader, bearerPrefix)
		info, ok := validTokens.Get(token)
		if !ok || info == nil {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("invalid or expired token"))
			return
		}
		resp := map[string]any{
			"username":  info.Username,
			"expiresAt": info.ExpiresAt.Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})))

	// Health check (no authentication required)
	mux.HandleFunc("/health", healthCheckHandler)

	handlerWithLogging := loggingHandler(mux)

	log.Printf("MCP server listening on %s", url)
	modeStr := "session"
	if useStateless {
		modeStr = "stateless"
	}
	log.Printf("Mode: %s (MCP_STATELESS=%v)", modeStr, useStateless)
	log.Printf("Tool: cityTime (cities: nyc, sf, boston)")
	log.Printf("OAuth login: http://%s/oauth/login", url)
	log.Printf("Root info: http://%s/", url)
	log.Printf("MCP endpoint: http://%s/mcp", url)
	log.Printf("REST time endpoint: http://%s/time?city=nyc", url)
	log.Printf("WhoAmI endpoint: http://%s/whoami", url)
	log.Printf("Health: http://%s/health", url)
	if !useStateless {
		log.Printf("Sessionful mode: For curl testing set MCP_STATELESS=1 and restart, or send a proper session initialization JSON-RPC request first.")
	} else {
		log.Printf("Stateless mode: Direct tools/list and tools/call POSTs are accepted.")
	}

	// Start the HTTP server with logging handler.
	if err := http.ListenAndServe(url, handlerWithLogging); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// bearerTokenMiddleware validates bearer tokens before allowing access to the MCP handler
func bearerTokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract bearer token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			log.Printf("Request rejected: missing Authorization header from %s", r.RemoteAddr)
			return
		}

		// Check for "Bearer " prefix
		const bearerPrefix = "Bearer "
		if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			log.Printf("Request rejected: invalid Authorization header format from %s", r.RemoteAddr)
			return
		}

		token := authHeader[len(bearerPrefix):]

		// Validate the token
		if err := ValidateBearerToken(token); err != nil {
			http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
			log.Printf("Request rejected: %v from %s", err, r.RemoteAddr)
			return
		}

		// Token is valid, proceed to the MCP handler
		next.ServeHTTP(w, r)
	})
}

func runClient(url string) {
	// Mark runClient unused silence by referencing conditionally
	if false {
		log.Printf("runClient not invoked: %s", url)
	}

	ctx := context.Background()

	// Create the URL for the server.
	log.Printf("Connecting to MCP server at %s", url)

	// Get bearer token from environment variable
	bearerToken := os.Getenv("MCP_BEARER_TOKEN")
	if bearerToken == "" {
		log.Println("Warning: MCP_BEARER_TOKEN not set. Authentication may fail.")
		log.Printf("To authenticate, visit: %s/oauth/login", url)
	}

	// Create an MCP client.
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "time-client",
		Version: "1.0.0",
	}, nil)

	// Create OAuth HTTP client with bearer token
	httpClient := CreateOAuthHTTPClient(bearerToken)

	// Connect to the server with authenticated HTTP client.
	// Note: Connect to /mcp endpoint instead of root
	mcpEndpoint := url + "/mcp"
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   mcpEndpoint,
		HTTPClient: httpClient,
	}, nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer session.Close()

	log.Printf("Connected to server (session ID: %s)", session.ID())

	// First, list available tools.
	log.Println("Listing available tools...")
	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	for _, tool := range toolsResult.Tools {
		log.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}

	// Call the cityTime tool for each city.
	cities := []string{"nyc", "sf", "boston"}

	log.Println("Getting time for each city...")
	for _, city := range cities {
		// Call the tool.
		result, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "cityTime",
			Arguments: map[string]any{
				"city": city,
			},
		})
		if err != nil {
			log.Printf("Failed to get time for %s: %v\n", city, err)
			continue
		}

		// Print the result.
		for _, content := range result.Content {
			if textContent, ok := content.(*mcp.TextContent); ok {
				log.Printf("  %s", textContent.Text)
			}
		}
	}

	log.Println("Client completed successfully")
}
