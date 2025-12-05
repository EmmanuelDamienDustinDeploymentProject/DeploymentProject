# DeploymentProject

![deploy](https://github.com/EmmanuelDamienDustinDeploymentProject/DeploymentProject/actions/workflows/deploy.yml/badge.svg?branch=main)
![go-build-lint-test](https://github.com/EmmanuelDamienDustinDeploymentProject/DeploymentProject/actions/workflows/go-build-lint-test.yml/badge.svg?branch=main)
![iac-linting](https://github.com/EmmanuelDamienDustinDeploymentProject/DeploymentProject/actions/workflows/iac-linting.yml/badge.svg?branch=main)
![trivy](https://github.com/EmmanuelDamienDustinDeploymentProject/DeploymentProject/actions/workflows/trivy.yml/badge.svg?branch=main)

## Usage
```bash
go run .
```
## Endpoints

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
      "type": "http",
      "url": "http://localhost:8080",
    }
  }
}
```

### Available Tools

- **get_city_time**: Get current time for NYC, SF, or Boston
- **get_fortune**: Get a random fortune message
- **apr**: Calculate APR (Annual Percentage Rate) for loans

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

## Deployment

### Setup Github OAuth Environment Variables

There isn't a terraform resource to create the Github OAuth app, so it has to be done manually.

Follow these instructions, and then use the command to update the environment variables and re-deploy the ECS service:

https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/creating-an-oauth-app

```
export GITHUB_CLIENT_ID="your_github_client_id"
export GITHUB_CLIENT_SECRET="your_github_client_secret"

# Get current task definition, update env vars, register new version, and deploy
TASK_DEF=$(aws ecs describe-task-definition --task-definition $(aws ecs list-task-definitions --family-prefix mcp-server --sort DESC --max-items 1 --query 'taskDefinitionArns[0]' --output text | awk -F'/' '{print $NF}') --region us-east-1 --query 'taskDefinition' | jq --arg cid "$GITHUB_CLIENT_ID" --arg csec "$GITHUB_CLIENT_SECRET" '.containerDefinitions[0].environment |= map(if .name == "GITHUB_CLIENT_ID" then .value = $cid elif .name == "GITHUB_CLIENT_SECRET" then .value = $csec else . end) | del(.taskDefinitionArn, .revision, .status, .requiresAttributes, .compatibilities, .registeredAt, .registeredBy)')

# Reg
aws ecs register-task-definition --cli-input-json "$TASK_DEF" --region us-east-1

aws ecs update-service --cluster mcp-cluster --service mcp-service --task-definition $(aws ecs list-task-definitions --family-prefix mcp-server --sort DESC --max-items 1 --query 'taskDefinitionArns[0]' --output text | awk -F'/' '{print $NF}') --force-new-deployment --region us-east-1
```

## Development

### MCP Inspector

The MCP Inspector is an interactive developer tool for testing and debugging MCP servers.

Youy can run [MCP Inspector](https://modelcontextprotocol.io/docs/tools/inspector) like this:
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
cd iac
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