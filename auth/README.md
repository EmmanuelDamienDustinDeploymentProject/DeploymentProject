# OAuth 2.1 Authentication Package

This package implements OAuth 2.1 authentication for the MCP server using GitHub as the authorization server.

## Architecture

### Components

1. **Models** (`models.go`)
   - Data structures for OAuth flows
   - RFC-compliant metadata types
   - Client registration request/response models

2. **Configuration** (`config.go`)
   - Environment-based configuration
   - Server settings and OAuth parameters
   - Validation and helper methods

3. **Storage** (`storage.go`)
   - Client registration storage
   - Token validation caching
   - In-memory implementations (production should use persistent storage)

4. **PKCE** (`pkce.go`)
   - Proof Key for Code Exchange implementation
   - Required by OAuth 2.1 for all clients
   - S256 challenge method support

## Key Features

### Protected Resource Metadata (RFC 9728)
- Endpoint: `/.well-known/oauth-protected-resource`
- Tells clients where to find authorization information
- Lists supported scopes and authorization servers

### Dynamic Client Registration (RFC 7591)
- Endpoint: `/register`
- Allows clients to register without user interaction
- Returns client credentials for OAuth flows

### Token Validation
- Validates Bearer tokens from Authorization header
- Integrates with GitHub API for user verification
- Supports audience/resource indicators (RFC 8707)

### VS Code Integration
- Compatible with VS Code's built-in GitHub authentication
- Supports required redirect URIs:
  - `http://127.0.0.1:33418` (local)
  - `https://vscode.dev/redirect` (web)

## Environment Variables

```bash
# Server Configuration
MCP_SERVER_URL=https://your-server.com
HOST=0.0.0.0
PORT=8080
USE_HTTPS=false

# GitHub OAuth App (required)
GITHUB_CLIENT_ID=your_client_id
GITHUB_CLIENT_SECRET=your_client_secret

# OAuth Settings
OAUTH_REDIRECT_URIS=http://127.0.0.1:33418,https://vscode.dev/redirect
OAUTH_SCOPES_SUPPORTED=mcp:tools,mcp:resources,read:user
TOKEN_EXPIRY_SECONDS=3600

# Security
ENFORCE_HTTPS=true
ENABLE_DCR=true
ALLOW_PUBLIC_CLIENTS=true

# GitHub API (optional, for GitHub Enterprise)
GITHUB_API_URL=https://api.github.com
GITHUB_AUTH_URL=https://github.com/login/oauth/authorize
GITHUB_TOKEN_URL=https://github.com/login/oauth/access_token
```

## Usage Example

```go
package main

import (
    "log"
    "EmmanuelDamienDustinDeploymentProject/DeploymentProject/auth"
)

func main() {
    // Load configuration from environment
    config, err := auth.LoadConfigFromEnv()
    if err != nil {
        log.Fatal(err)
    }

    // Validate configuration
    if err := config.Validate(); err != nil {
        log.Fatal(err)
    }

    // Create storage backends
    clientStorage := auth.NewInMemoryClientStorage()
    tokenCache := auth.NewInMemoryTokenCache()

    // Use in your HTTP handlers
    // (middleware implementation to be added in Phase 2)
}
```

## Security Considerations

### Token Validation
- Always validate tokens with GitHub API
- Check audience/resource indicators
- Verify token expiration
- Cache validation results to reduce API calls

### Client Registration
- Validate redirect URIs strictly
- Use secure random generation for client credentials
- Hash client secrets before storage
- Implement rate limiting for registration endpoint

### PKCE (Required)
- All clients must use PKCE per OAuth 2.1
- Only S256 challenge method is recommended
- Verify code_verifier matches code_challenge

### HTTPS Enforcement
- Production must use HTTPS (except localhost)
- Validate redirect URIs match registered values
- Implement state parameter verification

## Compliance

This implementation follows:
- OAuth 2.1 (draft-ietf-oauth-v2-1-13)
- RFC 7591 (Dynamic Client Registration)
- RFC 9728 (Protected Resource Metadata)
- RFC 8414 (Authorization Server Metadata)
- RFC 8707 (Resource Indicators)
- RFC 7636 (PKCE)
- MCP Authorization Specification (2025-06-18)

## Testing

See `auth_test` package for comprehensive test coverage including:
- Configuration validation
- PKCE generation and verification
- Client storage operations
- Token cache functionality
- Mock GitHub API responses

## Production Considerations

### Persistent Storage
Replace in-memory storage with:
- PostgreSQL/MySQL for client registrations
- Redis for token caching
- Encrypted storage for secrets

### Monitoring
- Log all authentication attempts
- Track failed validations
- Monitor token usage patterns
- Alert on suspicious activity

### Scalability
- Implement distributed token cache
- Use connection pooling for database
- Consider JWT tokens for stateless validation
- Implement rate limiting per client
