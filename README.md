# DeploymentProject

![deploy](https://github.com/EmmanuelDamienDustinDeploymentProject/DeploymentProject/actions/workflows/deploy.yml/badge.svg?branch=main)
![go-build-lint-test](https://github.com/EmmanuelDamienDustinDeploymentProject/DeploymentProject/actions/workflows/go-build-lint-test.yml/badge.svg?branch=main)
![iac-linting](https://github.com/EmmanuelDamienDustinDeploymentProject/DeploymentProject/actions/workflows/iac-linting.yml/badge.svg?branch=main)
![trivy](https://github.com/EmmanuelDamienDustinDeploymentProject/DeploymentProject/actions/workflows/trivy.yml/badge.svg?branch=main)

## Usage
```bash
go run .
```

The server will start with OAuth authentication enabled. Available endpoints:

- `/` - Protected MCP endpoint (requires OAuth token)
- `/health` - Health check (public)
- `/.well-known/oauth-protected-resource` - Protected resource metadata (public)
- `/.well-known/oauth-authorization-server` - Authorization server metadata (public)
- `/register` - Dynamic Client Registration (public, if DCR enabled)

## Usage
### MCP Client Configuration

Add the MCP server to your client configuration with OAuth support:

```json
{
  "servers": {
    "deployment-project": {
      "url": "http://localhost:8080",
      "auth": {
        "type": "oauth",
        "authorization_endpoint": "https://github.com/login/oauth/authorize",
        "token_endpoint": "https://github.com/login/oauth/access_token",
        "client_id": "your_github_client_id",
        "scopes": ["read:user"]
      }
    }
  }
}
```

### Available Tools

- **get_city_time**: Get current time for NYC, SF, or Boston
- **get_fortune**: Get a random fortune message
- **apr**: Calculate APR (Annual Percentage Rate) for loans

## OAuth 2.1 Implementation

This server implements OAuth 2.1 with the following features:

### Architecture

- **Authorization Server**: GitHub (github.com)
- **Resource Server**: This MCP server
- **Token Verification**: GitHub API validation with caching
- **Client Storage**: In-memory storage (suitable for development)
- **PKCE**: SHA-256 code challenge method (S256)

### Security Features

- Bearer token authentication using MCP SDK's `auth.RequireBearerToken`
- GitHub API token verification
- Scope validation (mcp:tools, mcp:resources)
- Audience/resource indicator validation (RFC 8707)
- HTTPS enforcement (configurable, disabled for localhost)
- Secure client secret storage (SHA-256 hashing)
- Token result caching to reduce GitHub API calls

### RFC Compliance

- [RFC 9728](https://datatracker.ietf.org/doc/html/rfc9728): OAuth 2.0 Protected Resource Metadata
- [RFC 8414](https://datatracker.ietf.org/doc/html/rfc8414): OAuth 2.0 Authorization Server Metadata  
- [RFC 7591](https://datatracker.ietf.org/doc/html/rfc7591): OAuth 2.0 Dynamic Client Registration
- [RFC 8707](https://datatracker.ietf.org/doc/html/rfc8707): Resource Indicators
- [RFC 7636](https://datatracker.ietf.org/doc/html/rfc7636): PKCE
- [OAuth 2.1 Draft](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-13)

### Environment Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_SERVER_URL` | Server's canonical URL | (required) |
| `GITHUB_CLIENT_ID` | GitHub OAuth App Client ID | (required) |
| `GITHUB_CLIENT_SECRET` | GitHub OAuth App Client Secret | (required) |
| `ENABLE_DCR` | Enable Dynamic Client Registration | `true` |
| `ALLOW_PUBLIC_CLIENTS` | Allow clients without secrets | `true` |
| `ENFORCE_HTTPS` | Require HTTPS (except localhost) | `false` |
| `TOKEN_EXPIRY_SECONDS` | Token cache expiry duration | `3600` |
| `OAUTH_SCOPES_SUPPORTED` | Comma-separated scopes | `mcp:tools,mcp:resources,read:user` |
| `OAUTH_REDIRECT_URIS` | Comma-separated redirect URIs | `http://127.0.0.1:33418,https://vscode.dev/redirect` |

### Fallback Mode

If OAuth configuration is invalid or missing, the server automatically runs in fallback mode without authentication. This allows for:
- Development without GitHub OAuth App setup
- Testing MCP functionality
- Gradual migration to authenticated deployment

## Development

### Testing with OAuth

#### Manual OAuth Flow Testing

The MCP Inspector may attempt to auto-discover and initiate OAuth flows. For simpler testing:

1. **Get a GitHub Personal Access Token**:
   - Go to https://github.com/settings/tokens
   - Click "Generate new token" → "Generate new token (classic)"
   - Select scopes: `read:user` (minimum required)
   - Copy the generated token

2. **Test the protected endpoint** with your token:
```bash
TOKEN="your_github_token_here"
curl -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' \
  http://localhost:8080/
```

3. **Test without authentication** (should return 401):
```bash
curl -v -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1}' \
  http://localhost:8080/
```

The response should include a `WWW-Authenticate` header with the resource metadata URL.

4. **Verify metadata endpoints**:
```bash
curl http://localhost:8080/.well-known/oauth-protected-resource | jq .
curl http://localhost:8080/.well-known/oauth-authorization-server | jq .
```

#### OAuth Flow with Browser

For full OAuth flow testing:

1. The authorization endpoint is at `http://localhost:8080/oauth/authorize`
2. It will redirect to GitHub for user authorization
3. GitHub redirects back to `http://localhost:8080/oauth/callback`
4. The token exchange happens at `http://localhost:8080/oauth/token` (proxied to GitHub)

**Note**: Your GitHub OAuth App must have `http://localhost:8080/oauth/callback` configured as an authorized callback URL.

#### Testing with VS Code

The server is designed to work with VS Code's built-in GitHub authentication. To test:

1. Configure your VS Code MCP settings to include the OAuth configuration shown in the [MCP Client Configuration](#mcp-client-configuration) section
2. VS Code will handle the OAuth flow with GitHub automatically
3. The server will verify the GitHub token on each request

### MCP Inspector

The MCP Inspector is an interactive developer tool for testing and debugging MCP servers.

https://modelcontextprotocol.io/docs/tools/inspector

**Note**: The MCP Inspector currently expects the resource server to also be the authorization server. Since this implementation uses GitHub as the authorization server, you'll need to provide a GitHub token manually for testing.

After starting the server and obtaining a GitHub token:
```bash
npx @modelcontextprotocol/inspector@0.16.7 --config mcp-inspector-config.json 
```

### Linting


#### Go

You can run [golangci-lint](https://golangci-lint.run/docs/welcome/install/#local-installation) with Docker like this:
```bash
docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v2.6.2 golangci-lint run
```

#### Terraform

##### terraform fmt and validate
```bash
terraform fmt -write=true --recursive
terraform validate
```

You can run [tflint](https://github.com/terraform-linters/tflint) with Docker like this:
```bash
docker run --rm -v $(pwd)/iac:/data -t --entrypoint /bin/sh ghcr.io/terraform-linters/tflint -c "tflint --init && tflint --recursive"
```

#### Docker

You can run [hadolint](https://github.com/hadolint/hadolint) with Docker like this:
```bash
docker run --rm -i hadolint/hadolint < Dockerfile
```

#### Security scanning with Trivy

You can run [trivy](https://trivy.dev/) with Docker like this:
```bash
docker run --rm -v "$(pwd):/workspace" aquasec/trivy fs \
  --ignore-unfixed \
  --severity CRITICAL,HIGH,MEDIUM,LOW \
  --scanners vuln,secret,misconfig \
  /workspace
```


## Libraries

### Go Standard Library
https://pkg.go.dev/net/http
https://pkg.go.dev/log
https://pkg.go.dev/fmt
https://pkg.go.dev/context
https://pkg.go.dev/time

### MCP SDK
https://github.com/modelcontextprotocol/go-sdk

### OAuth Dependencies
https://pkg.go.dev/golang.org/x/oauth2
https://pkg.go.dev/github.com/google/uuid

### Infrastructure
https://registry.terraform.io/providers/hashicorp/aws/latest/docs
https://registry.terraform.io/providers/integrations/github/latest/docs
https://docs.aws.amazon.com/AmazonECS/latest/developerguide/Welcome.html

### Linting Tools
https://golangci-lint.run/
https://github.com/terraform-linters/tflint
https://github.com/hadolint/hadolint

## Information

### MCP Documentation
https://github.com/modelcontextprotocol/go-sdk/tree/main/docs
https://modelcontextprotocol.io/docs/getting-started/intro
https://modelcontextprotocol.io/tutorials/building-mcp-with-llms
https://modelcontextprotocol.io/specification/2025-06-18
https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization

### VS Code Integration
https://code.visualstudio.com/docs/copilot/customization/mcp-servers
https://code.visualstudio.com/api/extension-guides/ai/mcp

### MCP Resources
https://github.com/microsoft/mcp-for-beginners
https://github.com/mcp
https://github.com/modelcontextprotocol/servers

### OAuth 2.1 Specifications
- [OAuth 2.1 Draft](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-13)
- [RFC 9728 - Protected Resource Metadata](https://datatracker.ietf.org/doc/html/rfc9728)
- [RFC 8414 - Authorization Server Metadata](https://datatracker.ietf.org/doc/html/rfc8414)
- [RFC 7591 - Dynamic Client Registration](https://datatracker.ietf.org/doc/html/rfc7591)
- [RFC 8707 - Resource Indicators](https://datatracker.ietf.org/doc/html/rfc8707)
- [RFC 7636 - PKCE](https://datatracker.ietf.org/doc/html/rfc7636)

## Project Structure

```
.
├── main.go                    # Main server with OAuth integration
├── auth/                      # OAuth 2.1 implementation package
│   ├── README.md             # Auth package documentation
│   ├── config.go             # OAuth configuration management
│   ├── models.go             # OAuth data structures
│   ├── storage.go            # Client and token storage
│   ├── pkce.go               # PKCE implementation
│   ├── github.go             # GitHub token verifier
│   ├── metadata.go           # Metadata endpoints (RFC 9728, RFC 8414)
│   ├── registration.go       # Dynamic Client Registration (RFC 7591)
│   └── middleware.go         # OAuth middleware integration
├── tools/                     # MCP tools
│   ├── get_city_time.go
│   ├── get_fortune.go
│   ├── get_apr.go
│   └── register_tools.go
├── iac/                       # Terraform infrastructure
│   └── modules/aws-ecs/
├── Dockerfile
└── README.md
```