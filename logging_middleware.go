// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Header() http.Header {
	return rw.ResponseWriter.Header()
}

func loggingHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code.
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Log request details including session ID if present
		sessionID := r.Header.Get("Mcp-Session-Id")
		sessionInfo := ""
		if sessionID != "" {
			sessionInfo = " | Session: " + sessionID
		}

		log.Printf("[REQUEST] %s | %s | %s %s%s",
			start.Format(time.RFC3339),
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			sessionInfo)

		// Call the actual handler.
		handler.ServeHTTP(wrapped, r)

		// Log response details including session ID if set in response
		responseSessionID := wrapped.Header().Get("Mcp-Session-Id")
		responseSessionInfo := ""
		if responseSessionID != "" {
			responseSessionInfo = " | Response Session: " + responseSessionID
		}

		duration := time.Since(start)
		log.Printf("[RESPONSE] %s | %s | %s %s | Status: %d | Duration: %v%s",
			time.Now().Format(time.RFC3339),
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration,
			responseSessionInfo)
	})
}
