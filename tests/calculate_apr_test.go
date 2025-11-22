package tests

import (
	"context"
	"testing"
	"encoding/json"
	"strings"

	"EmmanuelDamienDustinDeploymentProject/DeploymentProject/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCalculateAPR(t *testing.T) {
	tool := tools.CalculateAPR{}

	result, _, err := tool.Action(
		context.TODO(),
		&mcp.CallToolRequest{},
		&tools.CalculateAPRParams{
			Principal: 1000,
			TotalInterest: 10,
			TermInYears: 10,
		},
	)

	if err != nil {
		t.Errorf("Calling tool \"%s\" resulted in an error: %s", tool.Name, err)
	}

	var data map[string]interface{}
	jsonBytes, _ := result.Content[0].MarshalJSON()
	json.Unmarshal(jsonBytes, &data)

	splitResponse := strings.Split(data["text"].(string), " ")
	apr := strings.TrimSuffix(splitResponse[len(splitResponse)-1], ".")

	if apr != "0.10%" {
		t.Errorf("Calling tool \"%s\" resulted in an incorrect calculation, expected 0.10%% but got %s", tool.Name, apr)
	}
}
