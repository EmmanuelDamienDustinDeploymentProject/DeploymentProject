package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetCityTime struct{
	Name string
	Description string
}

// GetTimeParams defines the parameters for the cityTime tool.
type GetCityTimeParams struct {
	City string `json:"city" jsonschema:"City to get time for (nyc, sf, or boston)"`
}

// getTime implements the tool that returns the current time for a given city.
func (tool *GetCityTime) Action(ctx context.Context, req *mcp.CallToolRequest, params *GetCityTimeParams) (*mcp.CallToolResult, any, error) {
	// Define time zones for each city
	locations := map[string]string{
		"nyc":    "America/New_York",
		"sf":     "America/Los_Angeles",
		"boston": "America/New_York",
	}

	city := params.City
	if city == "" {
		city = "nyc" // Default to NYC
	}

	// Get the timezone.
	tzName, ok := locations[city]
	if !ok {
		return nil, nil, fmt.Errorf("unknown city: %s", city)
	}

	// Load the location.https://aphorismcookie.herokuapp.com
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load timezone: %w", err)
	}

	// Get current time in that location.
	now := time.Now().In(loc)

	// Format the response.
	cityNames := map[string]string{
		"nyc":    "New York City",
		"sf":     "San Francisco",
		"boston": "Boston",
	}

	response := fmt.Sprintf("The current time in %s is %s",
		cityNames[city],
		now.Format(time.RFC3339))

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: response},
		},
	}, nil, nil
}

func (tool *GetCityTime) Register(server *mcp.Server) (mcpToolInstance *mcp.Tool) {
	mcpToolInstance = &mcp.Tool{
		Name: tool.Name,
		Description: tool.Description,
	}

	mcp.AddTool(server, mcpToolInstance, tool.Action)

	return
}

func init() {
	tools = append(tools, &GetCityTime{
		Name: "get-city-time",
		Description: "Get the current time in NYC, San Francisco, or Boston",
	})
}
