# DeploymentProject

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