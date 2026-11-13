package routing_test

import (
	"testing"

	"aihub/internal/routing"
)

func TestProjectRequirementsMapsJSONSchema(t *testing.T) {
	status, err := routing.ProjectRequirements([]byte(`{"text":{"format":{"type":"json_schema"}}}`), routing.StructuralRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(status.RequiredCapabilities, "structured_output") {
		t.Fatalf("capabilities = %v", status.RequiredCapabilities)
	}
}

func TestProjectRequirementsMapsJSONObject(t *testing.T) {
	status, err := routing.ProjectRequirements([]byte(`{"text":{"format":{"type":"json_object"}}}`), routing.StructuralRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(status.RequiredCapabilities, "structured_output") {
		t.Fatalf("capabilities = %v", status.RequiredCapabilities)
	}
}

func TestProjectRequirementsCodexStyleToolsOmitCustomToolsGate(t *testing.T) {
	body := []byte(`{
		"tools": [
			{"type": "function", "name": "read_file"},
			{"type": "custom", "name": "apply_patch"}
		],
		"input": "hi"
	}`)
	status, err := routing.ProjectRequirements(body, routing.StructuralRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(status.RequiredCapabilities, "function_tools") {
		t.Fatalf("expected function_tools, got %v", status.RequiredCapabilities)
	}
	if contains(status.RequiredCapabilities, "custom_tools") {
		t.Fatalf("custom_tools must not gate routing, got %v", status.RequiredCapabilities)
	}
}

func TestProjectRequirementsAdditionalToolsRequireFunctionTools(t *testing.T) {
	body := []byte(`{
		"input":[
			{"type":"additional_tools","tools":[{"type":"function","name":"exec"}]},
			{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]}
		]
	}`)
	status, err := routing.ProjectRequirements(body, routing.StructuralRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(status.RequiredCapabilities, "function_tools") {
		t.Fatalf("expected function_tools from additional_tools, got %v", status.RequiredCapabilities)
	}
}

func TestProjectRequirementsFileInputUnmapped(t *testing.T) {
	status, err := routing.ProjectRequirements([]byte(`{"input":[{"type":"input_file"}]}`), routing.StructuralRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Unmapped {
		t.Fatal("expected unmapped requirement")
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}
