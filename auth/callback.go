// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"fmt"
	"log"
	"net/http"
)

// CallbackHandler handles OAuth callbacks from GitHub
type CallbackHandler struct {
	config *Config
}

// NewCallbackHandler creates a new callback handler
func NewCallbackHandler(config *Config) *CallbackHandler {
	return &CallbackHandler{
		config: config,
	}
}

// ServeHTTP implements http.Handler
func (h *CallbackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get the authorization code and state from the query parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")
	errorDescription := r.URL.Query().Get("error_description")

	// Check for errors from GitHub
	if errorParam != "" {
		http.Error(w, fmt.Sprintf("Authorization error: %s - %s", errorParam, errorDescription), http.StatusBadRequest)
		return
	}

	// Check if we have a code
	if code == "" {
		http.Error(w, "No authorization code received", http.StatusBadRequest)
		return
	}

	// Return HTML that will pass the code back to the opener window (for MCP Inspector)
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>Authorization Complete</title>
    <script>
        // Send the authorization code back to the opener window
        if (window.opener) {
            window.opener.postMessage({
                type: 'oauth-callback',
                code: '%s',
                state: '%s'
            }, '*');
            window.close();
        } else {
            document.getElementById('message').textContent = 'Authorization successful! You can close this window.';
        }
    </script>
</head>
<body>
    <h1 id="message">Processing authorization...</h1>
    <p>Code: %s</p>
    <p>If this window doesn't close automatically, you can close it manually.</p>
</body>
</html>`, code, state, code)

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(html)); err != nil {
		log.Printf("Failed to write callback response: %v", err)
	}
}
