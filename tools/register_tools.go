package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type MCPRegisterableTool interface {
	Register(server *mcp.Server)
} 

var tools []MCPRegisterableTool

func RegisterAll(server *mcp.Server) {
	for _, tool := range tools {
		tool.Register(server)
	}
}
