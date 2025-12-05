package tests

import (
	"context"
	"testing"
	"encoding/json"

	"EmmanuelDamienDustinDeploymentProject/DeploymentProject/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestGetFortune(t *testing.T) {
	tool := tools.GetFortune{}

	result, _, err := tool.Action(
		context.TODO(),
		&mcp.CallToolRequest{},
		&struct{}{},
	)

	if err != nil {
		t.Errorf("Calling tool \"%s\" resulted in an error: %s", tool.Name, err)
	}

	var data map[string]interface{}

	jsonBytes, _ := result.Content[0].MarshalJSON()
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(data["text"].(string)) < 1 {
		t.Errorf("Calling tool \"%s\" resulted in a response with 0 characters!", tool.Name)
	}
}
