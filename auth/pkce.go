// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// PKCE (Proof Key for Code Exchange) implementation per RFC 7636
// Required by OAuth 2.1 for all clients

const (
	// Code verifier must be 43-128 characters
	codeVerifierLength = 64 // Using 64 bytes for good entropy
)

// generateCodeVerifier generates a cryptographically random code verifier
// The verifier is a high-entropy cryptographic random string using the
// unreserved characters [A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~"
// with a minimum length of 43 characters and a maximum length of 128 characters
func generateCodeVerifier() (string, error) {
	// Generate random bytes
	b := make([]byte, codeVerifierLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	// Encode as base64url without padding
	verifier := base64.RawURLEncoding.EncodeToString(b)

	// Ensure length is within RFC requirements (43-128 characters)
	if len(verifier) < 43 {
		return "", fmt.Errorf("code verifier too short: %d characters", len(verifier))
	}
	if len(verifier) > 128 {
		verifier = verifier[:128]
	}

	return verifier, nil
}

// generateS256Challenge generates a code challenge from a code verifier using SHA256
// challenge = BASE64URL(SHA256(ASCII(code_verifier)))
func generateS256Challenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
