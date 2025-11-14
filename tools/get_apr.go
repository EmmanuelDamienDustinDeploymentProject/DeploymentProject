package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetAPR struct{}

// CalculateAPRParams defines the parameters for the apr tool.
type CalculateAPRParams struct {
	Principal     float64 `json:"principal" jsonschema:"The total loan amount (e.g., 10000)"`
	TotalInterest float64 `json:"totalInterest" jsonschema:"The total interest paid over the loan term (e.g., 1500)"`
	TermInYears   int     `json:"termInYears" jsonschema:"The loan term in years (e.g., 3)"`
}

func getAPR(ctx context.Context, req *mcp.CallToolRequest, params *CalculateAPRParams) (*mcp.CallToolResult, any, error) {
	if params.Principal <= 0 {
		return nil, nil, fmt.Errorf("principal must be greater than 0")
	}
	if params.TermInYears <= 0 {
		return nil, nil, fmt.Errorf("term in years must be greater than 0")
	}
	if params.TotalInterest < 0 {
		return nil, nil, fmt.Errorf("total interest cannot be negative")
	}

	totalInterestFraction := params.TotalInterest / params.Principal
	annualRateDecimal := totalInterestFraction / float64(params.TermInYears)
	apr := annualRateDecimal * 100
	response := fmt.Sprintf(
		"A loan of $%.2f with $%.2f total interest over %d years has a simple APR of %.2f%%.",
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

func (tool *GetAPR) Register(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "apr",
		Description: "Calculates the simple APR based on total interest paid.",
	}, getAPR)
}

func init() {
	tools = append(tools, &GetAPR{})
}
