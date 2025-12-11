package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const paymentsPerYear = 12.0

type CalculateAPR struct {
	Name        string
	Description string
}

// CalculateAPRParams defines the parameters for the apr tool.
type CalculateAPRParams struct {
	Principal     float64 `json:"principal" jsonschema:"The total loan amount (e.g., 10000)"`
	TotalInterest float64 `json:"totalInterest" jsonschema:"The total interest paid over the loan term (e.g., 1500)"`
	TermInYears   int     `json:"termInYears" jsonschema:"The loan term in years (e.g., 3)"`
}

func (tool *CalculateAPR) Action(ctx context.Context, req *mcp.CallToolRequest, params *CalculateAPRParams) (*mcp.CallToolResult, any, error) {
	if params.Principal <= 0 {
		return nil, nil, fmt.Errorf("principal must be greater than 0")
	}
	if params.TermInYears <= 0 {
		return nil, nil, fmt.Errorf("term in years must be greater than 0")
	}
	if params.TotalInterest < 0 {
		return nil, nil, fmt.Errorf("total interest cannot be negative")
	}

	totalPayments := float64(params.TermInYears) * paymentsPerYear

	numerator := 2.0 * params.TotalInterest * paymentsPerYear
	denominator := params.Principal * (totalPayments + 1.0)

	if denominator == 0 {
		return nil, nil, fmt.Errorf("invalid calculation resulting in zero denominator")
	}

	apr := (numerator / denominator) * 100

	response := fmt.Sprintf(
		"A loan of $%.2f with $%.2f total interest over %d years (monthly payments assumed) has an estimated APR of %.2f%%.",
		params.Principal,
		params.TotalInterest,
		params.TermInYears,
		apr,
	)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: response},
		},
	}, nil, nil
}

func (tool *CalculateAPR) Register(server *mcp.Server) (mcpToolInstance *mcp.Tool) {
	mcpToolInstance = &mcp.Tool{
		Name:        tool.Name,
		Description: tool.Description,
	}

	mcp.AddTool(server, mcpToolInstance, tool.Action)

	return
}

func init() {
	tools = append(tools, &CalculateAPR{
		Name:        "calculate-apr",
		Description: "Calculates the simple APR based on total interest paid.",
	})
}
