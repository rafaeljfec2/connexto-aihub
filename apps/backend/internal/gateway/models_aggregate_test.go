package gateway

import (
	"encoding/json"
	"testing"
)

func TestParseOpenAIModelListData(t *testing.T) {
	items, err := parseOpenAIModelListData([]byte(`{"object":"list","data":[{"id":"a"},{"id":"b"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len = %d, want 2", len(items))
	}
}

func TestParseOpenAIModelListDataEmpty(t *testing.T) {
	items, err := parseOpenAIModelListData([]byte(`{"object":"list","data":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if items != nil {
		t.Fatalf("items = %v, want nil slice", items)
	}
}

func TestParseOpenAIModelListDataInvalidJSON(t *testing.T) {
	_, err := parseOpenAIModelListData([]byte(`not json`))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMergedModelListPreservesEntries(t *testing.T) {
	raw := []byte(`{"object":"list","data":[{"id":"composer-2.5"},{"id":"deepseek-v4.1-flash"}]}`)
	var envelope struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	if len(envelope.Data) != 2 {
		t.Fatalf("len = %d", len(envelope.Data))
	}
}
