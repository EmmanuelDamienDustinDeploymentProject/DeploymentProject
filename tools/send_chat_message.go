package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"EmmanuelDamienDustinDeploymentProject/DeploymentProject/chat"
)

type SendChatMessage struct {
	Name        string
	Description string
	ChatServer  *chat.Server
}

// SendChatMessageParams defines the parameters for sending a chat message
type SendChatMessageParams struct {
	Message string `json:"message" jsonschema:"The message to send to the chat room"`
}

func (tool *SendChatMessage) Action(ctx context.Context, req *mcp.CallToolRequest, params *SendChatMessageParams) (*mcp.CallToolResult, any, error) {
	if params.Message == "" {
		return nil, nil, fmt.Errorf("message cannot be empty")
	}

	// Get the session ID from context (we'll set this up in middleware)
	sessionID, ok := ctx.Value("sessionID").(string)
	if !ok || sessionID == "" {
		return nil, nil, fmt.Errorf("no active session found")
	}

	// Get the connection to find the GitHub username
	conn, ok := tool.ChatServer.GetConnection(sessionID)
	if !ok {
		return nil, nil, fmt.Errorf("connection not found for session")
	}

	// Broadcast the message
	if err := tool.ChatServer.BroadcastMessage(conn.GitHubUser, params.Message); err != nil {
		return nil, nil, fmt.Errorf("failed to broadcast message: %w", err)
	}

	response := fmt.Sprintf("Message sent from %s", conn.GitHubUser)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: response},
		},
	}, nil, nil
}

func (tool *SendChatMessage) Register(server *mcp.Server) (mcpToolInstance *mcp.Tool) {
	mcpToolInstance = &mcp.Tool{
		Name:        tool.Name,
		Description: tool.Description,
	}

	mcp.AddTool(server, mcpToolInstance, tool.Action)

	return
}

func NewSendChatMessage(chatServer *chat.Server) *SendChatMessage {
	return &SendChatMessage{
		Name:        "send-chat-message",
		Description: "Send a message to the global chat room. All connected users will receive your message.",
		ChatServer:  chatServer,
	}
}
