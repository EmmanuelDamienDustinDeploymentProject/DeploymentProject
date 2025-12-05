package tests

import (
	"context"
	"testing"
	"encoding/json"
	"strings"

	"EmmanuelDamienDustinDeploymentProject/DeploymentProject/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestGetCityTimeInNYC(t *testing.T) {
	tool := tools.GetCityTime{}

	result, _, err := tool.Action(
		context.TODO(),
		&mcp.CallToolRequest{},
		&tools.GetCityTimeParams{
			City: "nyc",
		},
	)

	if err != nil {
		t.Errorf("Calling tool \"%s\" resulted in an error: %s", tool.Name, err)
	}

	var data map[string]interface{}
	jsonBytes, _ := result.Content[0].MarshalJSON()
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !strings.Contains(data["text"].(string), "New York City") {
		t.Errorf("Calling tool \"%s\" with \"nyc\" as the city returned the wrong data: %s", tool.Name, data["text"].(string))
	}
}

func TestGetCityTimeInBoston(t *testing.T) {
	tool := tools.GetCityTime{}

	result, _, err := tool.Action(
		context.TODO(),
		&mcp.CallToolRequest{},
		&tools.GetCityTimeParams{
			City: "boston",
		},
	)

	if err != nil {
		t.Errorf("Calling tool \"%s\" resulted in an error: %s", tool.Name, err)
	}

	var data map[string]interface{}
	jsonBytes, _ := result.Content[0].MarshalJSON()
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !strings.Contains(data["text"].(string), "Boston") {
		t.Errorf("Calling tool \"%s\" with \"nyc\" as the city returned the wrong data: %s", tool.Name, data["text"].(string))
	}
}

func TestGetCityTimeInSanFrancisco(t *testing.T) {
	tool := tools.GetCityTime{}

	result, _, err := tool.Action(
		context.TODO(),
		&mcp.CallToolRequest{},
		&tools.GetCityTimeParams{
			City: "sf",
		},
	)

	if err != nil {
		t.Errorf("Calling tool \"%s\" resulted in an error: %s", tool.Name, err)
	}

	var data map[string]interface{}
	jsonBytes, _ := result.Content[0].MarshalJSON()
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if !strings.Contains(data["text"].(string), "San Francisco") {
		t.Errorf("Calling tool \"%s\" with \"nyc\" as the city returned the wrong data: %s", tool.Name, data["text"].(string))
	}
}
