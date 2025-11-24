# DeploymentProject

## OAuth Setup

This MCP server supports OAuth 2.0 authentication with GitHub, implementing:
- RFC 8414 (OAuth 2.0 Authorization Server Metadata)
- RFC 9728 (OAuth 2.0 Protected Resource Metadata)
- RFC 7591 (Dynamic Client Registration)

**Authentication Mode**: Always required for tool calls. Discovery methods (initialize, tools/list, prompts/list, resources/list) work without authentication to enable OAuth flow initiation.

### Environment Variables

Set your GitHub OAuth app credentials:
```bash
export GITHUB_CLIENT_ID=your_client_id
export GITHUB_CLIENT_SECRET=your_client_secret
```

The server automatically initializes OAuth if these environment variables are set. If not set, the server runs without authentication (development mode).

### GitHub OAuth App Setup

1. Go to GitHub Settings > Developer settings > OAuth Apps
2. Create a new OAuth App with:
   - Application name: DeploymentProject MCP
   - Homepage URL: http://localhost:8080
   - Authorization callback URL: http://localhost:8080/oauth/callback (or any redirect URI you configure)

## Usage
```bash
go run .
```

Now, add the mcp server like this:
```json

{
  "servers": {
    "deployment-project": {
      "url": "http://localhost:8000"
    }
  }
}
```

And you can ask for the timezone for NYC, SF or Boston.

## Development

### MCP Inspector

The MCP Inspector is an interactive developer tool for testing and debugging MCP servers.

https://modelcontextprotocol.io/docs/tools/inspector

After starting the go server, run this npx command and once the browser opens click "Connect".
```bash
npx @modelcontextprotocol/inspector --config mcp-inspector-config.json 
```

### Linting

You can run [golangci-lint](https://golangci-lint.run/docs/welcome/install/#local-installation) with Docker like this:
```bash
docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v2.6.2 golangci-lint run
```

## Libraries

https://pkg.go.dev/net/http
https://pkg.go.dev/log
https://pkg.go.dev/fmt
https://pkg.go.dev/context
https://pkg.go.dev/time

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