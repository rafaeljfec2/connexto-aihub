package routing_test

import (
	"strings"
	"testing"

	"aihub/internal/routing"
)

func TestValidateStructureRejectsInvalidModelShapes(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{name: "null model", payload: `{"model":null}`},
		{name: "empty model", payload: `{"model":""}`},
		{name: "whitespace model", payload: `{"model":"   "}`},
		{name: "object model", payload: `{"model":{"id":"x"}}`},
		{name: "zero max_output_tokens", payload: `{"model":"m","max_output_tokens":0}`},
		{name: "negative max_output_tokens", payload: `{"model":"m","max_output_tokens":-1}`},
		{name: "string max_output_tokens", payload: `{"model":"m","max_output_tokens":"10"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := routing.ValidateStructure([]byte(test.payload))
			if err == nil {
				t.Fatal("expected structural error")
			}
		})
	}
}

func TestValidateStructureAllowsAbsentModel(t *testing.T) {
	got, err := routing.ValidateStructure([]byte(`{"input":"hello"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ModelPresent {
		t.Fatal("model should be absent")
	}
}

func TestValidateStructureDetectsDuplicateModelKey(t *testing.T) {
	_, err := routing.ValidateStructure([]byte(`{"model":"a","model":"b"}`))
	if err == nil {
		t.Fatal("expected duplicate model error")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateStructureModelTooLong(t *testing.T) {
	_, err := routing.ValidateStructure([]byte(`{"model":"` + strings.Repeat("m", 4097) + `"}`))
	if err == nil {
		t.Fatal("expected model too long")
	}
}
