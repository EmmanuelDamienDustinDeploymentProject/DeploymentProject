package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetFortune struct{
	Name string
	Description string
}

type FortuneAPIResponse struct {
	Data struct {
		Message string `json:"message"`
	} `json:"data"`
	Meta struct {
		Status string `json:"status"`
	} `json:"meta"`
}

func (tool *GetFortune) Action(ctx context.Context, req *mcp.CallToolRequest, params *struct{}) (*mcp.CallToolResult, any, error) {
	res, err := http.Get("https://aphorismcookie.herokuapp.com/")
	if err != nil {
		return nil, nil, fmt.Errorf("connecting to fortune API failed!: %s", err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("failed to close response body: %v\n", err)
		}
	}(res.Body)

	var resAsJSON FortuneAPIResponse
	err = json.NewDecoder(res.Body).Decode(&resAsJSON)
	if err != nil {
		fmt.Printf("failed to decode json in getFortune: %v\n", err)
		return nil, nil, err
	}

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
