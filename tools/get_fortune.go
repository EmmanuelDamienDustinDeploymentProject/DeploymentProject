package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetFortune struct{}

type FortuneAPIResponse struct {
	Data	struct{Message string `json:"message"`} `json:"data"`
	Meta	struct{Status string `json:"status"`} `json:"meta"`
}

func getFortune(ctx context.Context, req *mcp.CallToolRequest, params *struct{}) (*mcp.CallToolResult, any, error) {
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

func (tool *GetFortune) Register(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "Get Fortune",
		Description: "Gets a random fortune from aphorismcookie.herokuapp.com",
	}, getFortune)
}

func init() {
	tools = append(tools, &GetFortune{})
}
