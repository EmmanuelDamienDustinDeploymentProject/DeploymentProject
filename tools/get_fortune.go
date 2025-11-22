package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetFortune struct{
	Name string
	Description string
}

type FortuneAPIResponse struct {
	Data	struct{Message string `json:"message"`} `json:"data"`
	Meta	struct{Status string `json:"status"`} `json:"meta"`
}

func (tool *GetFortune) Action(ctx context.Context, req *mcp.CallToolRequest, params *struct{}) (*mcp.CallToolResult, any, error) {
	res, err := http.Get("https://aphorismcookie.herokuapp.com/")
	if err != nil {
		return nil, nil, fmt.Errorf( "Connecting to fortune API failed!: %s", err )
	}

	defer res.Body.Close()

	var resAsJSON FortuneAPIResponse
	json.NewDecoder(res.Body).Decode(&resAsJSON)

	fortune := resAsJSON.Data.Message

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fortune},
		},
	}, nil, nil
}

func (tool *GetFortune) Register(server *mcp.Server) (mcpToolInstance *mcp.Tool) {
	mcpToolInstance = &mcp.Tool{
		Name: tool.Name,
		Description: tool.Description,
	}

	mcp.AddTool(server, mcpToolInstance, tool.Action)

	return
}

func init() {
	tools = append(tools, &GetFortune{
		Name: "Get Fortune",
		Description: "Gets a random fortune from aphorismcookie.herokuapp.com",
	})
}
