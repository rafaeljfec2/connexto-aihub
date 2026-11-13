package gateway

import (
	"strings"
	"testing"

	"aihub/internal/providers"
)

func TestPromoteAdditionalToolsToTopLevel(t *testing.T) {
	in := []byte(`{
		"model":"gpt-6-luna",
		"tool_choice":"auto",
		"input":[
			{"type":"additional_tools","tools":[
				{"type":"namespace","name":"functions","tools":[
					{"type":"custom","name":"exec","description":"run"},
					{"type":"function","name":"probe_tool","parameters":{"type":"object","properties":{}}}
				]}
			]},
			{"type":"message","role":"user","content":[{"type":"input_text","text":"call probe_tool"}]}
		]
	}`)
	out, err := promoteAdditionalToolsToTopLevel(in)
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	text := string(out)
	if strings.Contains(text, `"additional_tools"`) {
		t.Fatalf("additional_tools should be removed: %s", text)
	}
	if !strings.Contains(text, `"name":"probe_tool"`) || !strings.Contains(text, `"name":"exec"`) {
		t.Fatalf("tools not promoted: %s", text)
	}
	if !strings.Contains(text, `"type":"message"`) {
		t.Fatalf("user message removed: %s", text)
	}
}

func TestAdaptOpenCodePromotesAdditionalTools(t *testing.T) {
	in := []byte(`{
		"model":"gpt-6-luna",
		"input":[
			{"type":"additional_tools","tools":[{"type":"function","name":"probe_tool","parameters":{"type":"object","properties":{}}}]},
			{"type":"input_text","text":"call probe_tool"}
		]
	}`)
	out, err := adaptResponsesPayloadForProvider(providers.OpenCodeGoID, in)
	if err != nil {
		t.Fatalf("adapt: %v", err)
	}
	text := string(out)
	if strings.Contains(text, `"additional_tools"`) {
		t.Fatal("additional_tools remained for OpenCode")
	}
	if !strings.Contains(text, `"name":"probe_tool"`) {
		t.Fatalf("probe_tool missing: %s", text)
	}
}
