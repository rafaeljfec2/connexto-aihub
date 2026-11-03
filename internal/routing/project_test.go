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
