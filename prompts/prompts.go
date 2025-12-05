package prompts

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RegisterAll registers all prompts with the MCP server
func RegisterAll(server *mcp.Server) {
	// APR Calculator prompt
	aprPrompt := &mcp.Prompt{
		Name:        "calculate-loan-apr",
		Description: "Calculate the Annual Percentage Rate (APR) for a loan",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "principal",
				Description: "The total loan amount in dollars",
				Required:    true,
			},
			{
				Name:        "total_interest",
				Description: "The total interest paid over the loan term in dollars",
				Required:    true,
			},
			{
				Name:        "term_years",
				Description: "The loan term in years",
				Required:    true,
			},
		},
	}

	server.AddPrompt(aprPrompt, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := req.Params.Arguments
		principal := args["principal"]
		totalInterest := args["total_interest"]
		termYears := args["term_years"]

		message := "Please calculate the APR for a loan with the following details:\n\n"
		message += "- Loan Amount (Principal): $" + principal + "\n"
		message += "- Total Interest Paid: $" + totalInterest + "\n"
		message += "- Loan Term: " + termYears + " years\n\n"
		message += "Use the calculate-apr tool to compute the annual percentage rate."

		return &mcp.GetPromptResult{
			Description: "APR calculation request",
			Messages: []*mcp.PromptMessage{
				{
					Role: "user",
					Content: &mcp.TextContent{
						Text: message,
					},
				},
			},
		}, nil
	})

	log.Printf("Registered prompt: %s", aprPrompt.Name)

	// City Time prompt
	timePrompt := &mcp.Prompt{
		Name:        "check-city-time",
		Description: "Get the current time in a major US city",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "city",
				Description: "The city name (nyc, sf, or boston)",
				Required:    true,
			},
		},
	}

	server.AddPrompt(timePrompt, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := req.Params.Arguments
		city := args["city"]

		message := "What is the current time in " + city + "?\n\n"
		message += "Use the get-city-time tool to retrieve the current local time."

		return &mcp.GetPromptResult{
			Description: "City time check request",
			Messages: []*mcp.PromptMessage{
				{
					Role: "user",
					Content: &mcp.TextContent{
						Text: message,
					},
				},
			},
		}, nil
	})

	log.Printf("Registered prompt: %s", timePrompt.Name)

	// Fortune prompt
	fortunePrompt := &mcp.Prompt{
		Name:        "get-daily-fortune",
		Description: "Get an inspirational fortune or aphorism",
		Arguments:   []*mcp.PromptArgument{},
	}

	server.AddPrompt(fortunePrompt, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		message := "Please get me a random fortune or inspirational quote.\n\n"
		message += "Use the get-fortune tool to retrieve an aphorism."

		return &mcp.GetPromptResult{
			Description: "Fortune retrieval request",
			Messages: []*mcp.PromptMessage{
				{
					Role: "user",
					Content: &mcp.TextContent{
						Text: message,
					},
				},
			},
		}, nil
	})

	log.Printf("Registered prompt: %s", fortunePrompt.Name)
}
