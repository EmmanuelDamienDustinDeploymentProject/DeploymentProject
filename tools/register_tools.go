package tools

import (
	"log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type MCPRegisterableTool interface {
	Register(server *mcp.Server) (mcpToolInstance *mcp.Tool)
} 

var tools []MCPRegisterableTool

func RegisterAll(server *mcp.Server) {
	for _, tool := range tools {
		mcpToolInstance := tool.Register(server)

		log.Printf("Registered tool: %s", mcpToolInstance.Name)
	}
}
