package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"EmmanuelDamienDustinDeploymentProject/DeploymentProject/chat"
)

type GetChatHistory struct {
	Name        string
	Description string
	ChatServer  *chat.Server
}

// GetChatHistoryParams defines the parameters for getting chat history
type GetChatHistoryParams struct {
	Limit int `json:"limit,omitempty" jsonschema:"Number of recent messages to retrieve (default: 20, max: 100)"`
}

func (tool *GetChatHistory) Action(ctx context.Context, req *mcp.CallToolRequest, params *GetChatHistoryParams) (*mcp.CallToolResult, any, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Get message history
	messages := tool.ChatServer.GetMessageHistory(limit)

	// Format messages
	var response string
	if len(messages) == 0 {
		response = "No messages in chat history."
	} else {
		response = fmt.Sprintf("Last %d messages:\n\n", len(messages))
		for _, msg := range messages {
			response += fmt.Sprintf("[%s] %s: %s\n",
				msg.Timestamp.Format("15:04:05"),
				msg.Sender,
				msg.Message,
			)
		}
	}

	// Also include as structured data
	jsonData, _ := json.MarshalIndent(messages, "", "  ")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: response},
			&mcp.TextContent{Text: fmt.Sprintf("\nStructured data:\n%s", string(jsonData))},
		},
	}, nil, nil
}

func (tool *GetChatHistory) Register(server *mcp.Server) (mcpToolInstance *mcp.Tool) {
	mcpToolInstance = &mcp.Tool{
		Name:        tool.Name,
		Description: tool.Description,
	}

	mcp.AddTool(server, mcpToolInstance, tool.Action)

	return
}

func NewGetChatHistory(chatServer *chat.Server) *GetChatHistory {
	return &GetChatHistory{
		Name:        "get-chat-history",
		Description: "Retrieve recent chat messages from the global chat room.",
		ChatServer:  chatServer,
	}
}
