package auth

// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

// ClientStorage defines the interface for storing and retrieving OAuth clients
type ClientStorage interface {
	// StoreClient stores a registered OAuth client
	StoreClient(client *OAuthClient) error

	// GetClient retrieves a client by client ID
	GetClient(clientID string) (*OAuthClient, error)

	// DeleteClient removes a client from storage
	DeleteClient(clientID string) error

	// ListClients returns all registered clients
	ListClients() ([]*OAuthClient, error)

	// ValidateClientSecret checks if the provided secret matches the stored client
	ValidateClientSecret(clientID, secret string) (bool, error)
}

// InMemoryClientStorage provides an in-memory implementation of ClientStorage
// This is suitable for development and testing, but should be replaced with
// persistent storage (database, Redis, etc.) for production use
type InMemoryClientStorage struct {
	mu      sync.RWMutex
	clients map[string]*OAuthClient
}

// NewInMemoryClientStorage creates a new in-memory client storage
func NewInMemoryClientStorage() *InMemoryClientStorage {
	return &InMemoryClientStorage{
		clients: make(map[string]*OAuthClient),
	}
}

// NewInMemoryClientStorageWithDefaults creates a new in-memory client storage
// with optional default clients for common MCP clients
func NewInMemoryClientStorageWithDefaults() *InMemoryClientStorage {
	storage := NewInMemoryClientStorage()
	
	// Pre-register a generic VS Code client with standard redirect URIs
	// This allows any VS Code instance to authenticate without explicit registration
	vsCodeClient := &OAuthClient{
		ClientID:     "vscode",
		ClientSecret: "", // Public client - no secret
		Metadata: ClientRegistrationRequest{
			RedirectURIs: []string{
				"http://127.0.0.1:33418",
				"http://127.0.0.1:33418/",
				"http://127.0.0.1:33418/done",
				"https://vscode.dev/redirect",
			},
			TokenEndpointAuthMethod: "none", // Public client
			GrantTypes: []string{
				"authorization_code",
			},
			ResponseTypes: []string{
				"code",
			},
			ClientName: "Visual Studio Code",
			Scope:      "mcp:tools mcp:resources read:user",
		},
		CreatedAt: time.Now(),
	}
	
	_ = storage.StoreClient(vsCodeClient)
	
	return storage
}

// StoreClient stores a registered OAuth client
func (s *InMemoryClientStorage) StoreClient(client *OAuthClient) error {
	if client == nil {
		return fmt.Errorf("client cannot be nil")
	}
	if client.ClientID == "" {
		return fmt.Errorf("client ID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a copy to avoid external modifications
	storedClient := *client
	s.clients[client.ClientID] = &storedClient

	return nil
}

// GetClient retrieves a client by client ID
func (s *InMemoryClientStorage) GetClient(clientID string) (*OAuthClient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, exists := s.clients[clientID]
	if !exists {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}

	// Return a copy to prevent external modifications
	clientCopy := *client
	return &clientCopy, nil
}

// DeleteClient removes a client from storage
func (s *InMemoryClientStorage) DeleteClient(clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clients[clientID]; !exists {
		return fmt.Errorf("client not found: %s", clientID)
	}
	delete(s.clients, clientID)

	return nil
}

// ListClients returns all registered clients
func (s *InMemoryClientStorage) ListClients() ([]*OAuthClient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := make([]*OAuthClient, 0, len(s.clients))
	for _, client := range s.clients {
		clientCopy := *client
		clients = append(clients, &clientCopy)
	}

	return clients, nil
}

// ValidateClientSecret checks if the provided secret matches the stored client
func (s *InMemoryClientStorage) ValidateClientSecret(clientID, secret string) (bool, error) {
	client, err := s.GetClient(clientID)
	if err != nil {
		return false, err
	}

	// Hash the provided secret and compare with stored hash
	hashedSecret := hashSecret(secret)
	return client.ClientSecret == hashedSecret, nil
}

// GenerateClientID generates a random client ID
func GenerateClientID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate client ID: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateClientSecret generates a random client secret
func GenerateClientSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate client secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashSecret hashes a client secret for secure storage
func hashSecret(secret string) string {
	hash := sha256.Sum256([]byte(secret))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// TokenCache defines the interface for caching token validation results
// This helps reduce calls to GitHub's API for frequently validated tokens
type TokenCache interface {
	// Set stores a token validation result with an expiry
	Set(token string, result *TokenValidationResult, expiry time.Duration) error

	// Get retrieves a cached token validation result
	Get(token string) (*TokenValidationResult, bool)

	// Delete removes a token from the cache
	Delete(token string) error
}

// InMemoryTokenCache provides an in-memory implementation of TokenCache
type InMemoryTokenCache struct {
	mu    sync.RWMutex
	cache map[string]*cacheEntry
}

type cacheEntry struct {
	result    *TokenValidationResult
	expiresAt time.Time
}

// NewInMemoryTokenCache creates a new in-memory token cache
func NewInMemoryTokenCache() *InMemoryTokenCache {
	cache := &InMemoryTokenCache{
		cache: make(map[string]*cacheEntry),
	}

	// Start background cleanup goroutine
	go cache.cleanupExpired()

	return cache
}

// Set stores a token validation result with an expiry
func (c *InMemoryTokenCache) Set(token string, result *TokenValidationResult, expiry time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[token] = &cacheEntry{
		result:    result,
		expiresAt: time.Now().Add(expiry),
	}

	return nil
}

// Get retrieves a cached token validation result
func (c *InMemoryTokenCache) Get(token string) (*TokenValidationResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[token]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.result, true
}

// Delete removes a token from the cache
func (c *InMemoryTokenCache) Delete(token string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, token)

	return nil
}

// cleanupExpired removes expired entries from the cache periodically
func (c *InMemoryTokenCache) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for token, entry := range c.cache {
			if now.After(entry.expiresAt) {
				delete(c.cache, token)
			}
		}
		c.mu.Unlock()
	}
}
