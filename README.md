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