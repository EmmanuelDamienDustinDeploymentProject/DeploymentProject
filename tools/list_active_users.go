package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"EmmanuelDamienDustinDeploymentProject/DeploymentProject/chat"
)

type ListActiveUsers struct {
	Name        string
	Description string
	ChatServer  *chat.Server
}

// ListActiveUsersParams defines the parameters (none needed)
type ListActiveUsersParams struct{}

func (tool *ListActiveUsers) Action(ctx context.Context, req *mcp.CallToolRequest, params *ListActiveUsersParams) (*mcp.CallToolResult, any, error) {
	// Get active users
	users := tool.ChatServer.GetActiveUsers()

	var response string
	if len(users) == 0 {
		response = "No users currently connected to chat."
	} else {
		response = fmt.Sprintf("Active users (%d):\n", len(users))
		response += "• " + strings.Join(users, "\n• ")
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: response},
		},
	}, nil, nil
}

func (tool *ListActiveUsers) Register(server *mcp.Server) (mcpToolInstance *mcp.Tool) {
	mcpToolInstance = &mcp.Tool{
		Name:        tool.Name,
		Description: tool.Description,
	}

	mcp.AddTool(server, mcpToolInstance, tool.Action)

	return
}

func NewListActiveUsers(chatServer *chat.Server) *ListActiveUsers {
	return &ListActiveUsers{
		Name:        "list-active-users",
		Description: "List all users currently connected to the global chat room.",
		ChatServer:  chatServer,
	}
}
