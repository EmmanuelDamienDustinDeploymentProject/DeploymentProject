# DeploymentProject

A Model Context Protocol (MCP) server with GitHub OAuth authentication.

## Features

- **MCP Server**: Time zone tool for NYC, SF, and Boston
- **GitHub OAuth**: Secure authentication using GitHub OAuth 2.0
- **Bearer Token Auth**: Server-side authentication via `RequireBearerToken`
- **Client OAuth Support**: Client-side OAuth via custom `http.Client`

## Setup

### 1. Create GitHub OAuth App

1. Go to GitHub Settings → Developer settings → OAuth Apps
2. Click "New OAuth App"
3. Fill in the details:
   - **Application name**: Your app name
   - **Homepage URL**: `http://localhost:8080` (or your domain)
   - **Authorization callback URL**: `http://localhost:8080/oauth/callback`
4. Copy the **Client ID** and generate a **Client Secret**

### 2. Configure Environment Variables

Copy the example environment file:
```bash
cp .env.example .env
```

Edit `.env` and add your GitHub OAuth credentials:
```bash
GITHUB_CLIENT_ID=your_github_client_id_here
GITHUB_CLIENT_SECRET=your_github_client_secret_here
OAUTH_REDIRECT_URL=http://localhost:8080/oauth/callback
```

Load environment variables:
```bash
export $(cat .env | xargs)
```

## Usage

### Start the Server

```bash
go run .
```

The server will start on `http://0.0.0.0:8080` with the following endpoints:
- `/mcp` - MCP endpoint (requires authentication)
- `/oauth/login` - Initiate OAuth flow
- `/oauth/callback` - OAuth callback handler
- `/health` - Health check (no auth required)

### Authenticate and Get Bearer Token

1. Open your browser and visit: `http://localhost:8080/oauth/login`
2. Authorize the app on GitHub
3. Copy the bearer token from the success page
4. Save the token to use with your MCP client

### Using with MCP Client

Configure your MCP client with the bearer token:

```json
{
  "servers": {
    "deployment-project": {
      "url": "http://localhost:8080/mcp",
      "headers": {
        "Authorization": "Bearer YOUR_TOKEN_HERE"
      }
    }
  }
}
```

Or set the environment variable:
```bash
export MCP_BEARER_TOKEN=your_token_here
```

### Using the Go Client

The repository includes a client example in `runClient()`:

```bash
export MCP_BEARER_TOKEN=your_token_here
# Modify main.go to call runClient instead of runServer
go run .
```

## Development

### MCP Inspector

The MCP Inspector is an interactive developer tool for testing and debugging MCP servers.

https://modelcontextprotocol.io/docs/tools/inspector

After starting the go server and authenticating, run this npx command and once the browser opens click "Connect":
```bash
npx @modelcontextprotocol/inspector --config mcp-inspector-config.json 
```

## Architecture

### OAuth Flow

1. User visits `/oauth/login`
2. Server redirects to GitHub OAuth
3. User authorizes the application
4. GitHub redirects to `/oauth/callback` with authorization code
5. Server exchanges code for access token
6. Server validates user with GitHub API
7. Server stores token and returns it to user
8. User includes token in `Authorization: Bearer <token>` header

### Authentication

- **Server-side**: Uses `RequireBearerToken` callback to validate tokens
- **Client-side**: Uses custom `http.Client` with `tokenTransport` to add bearer tokens
- **Token Storage**: In-memory store (use Redis/database in production)
- **Token Expiry**: 24 hours (configurable)

## Libraries

https://pkg.go.dev/net/http
https://pkg.go.dev/log
https://pkg.go.dev/fmt
https://pkg.go.dev/context
https://pkg.go.dev/time
https://pkg.go.dev/golang.org/x/oauth2

https://github.com/modelcontextprotocol/go-sdk

https://registry.terraform.io/providers/hashicorp/aws/latest/docs
https://registry.terraform.io/providers/integrations/github/latest/docs
https://docs.aws.amazon.com/AmazonECS/latest/developerguide/Welcome.html

## Information

https://github.com/modelcontextprotocol/go-sdk/tree/main/docs


https://modelcontextprotocol.io/docs/getting-started/intro
https://modelcontextprotocol.io/tutorials/building-mcp-with-llms
https://modelcontextprotocol.io/specification/2025-06-18

https://code.visualstudio.com/docs/copilot/customization/mcp-servers
https://code.visualstudio.com/api/extension-guides/ai/mcp
https://github.com/microsoft/mcp-for-beginners
https://github.com/mcp
https://github.com/modelcontextprotocol/servers
https://hub.docker.com/mcp