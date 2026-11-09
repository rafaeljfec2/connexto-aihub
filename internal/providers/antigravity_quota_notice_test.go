package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func noticeText(resets string) string {
	return "⚠ Individual quota reached. Please upgrade your subscription to increase your limits. Resets in " + resets + "."
}

func jsonString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func jsonResponsesDocument(text string) []byte {
	return []byte(`{"object":"response","model":"antigravity:claude-opus-4-6-thinking","output":[{"type":"message","content":[{"type":"output_text","text":` + jsonString(text) + `}]}]}`)
}

func response200JSON(body []byte) *http.Response {
	return &http.Response{
		StatusCode:    http.StatusOK,
		Header:        http.Header{"Content-Type": []string{"application/json"}},
		ContentLength: int64(len(body)),
		Body:          io.NopCloser(bytes.NewReader(body)),
	}
}

func response200SSE(events ...string) *http.Response {
	joined := strings.Join(events, "")
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(bytes.NewBufferString(joined)),
	}
}

func sseEvent(data string) string {
	return "event: response.output_text.delta\ndata: " + data + "\n\n"
}

func testContext() context.Context { return context.Background() }

func TestClassifyAntigravityJSONNoticeBlocks(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected time.Duration
	}{
		{
			name:     "plain notice",
			text:     noticeText("54h24m58s"),
			expected: 54*time.Hour + 24*time.Minute + 58*time.Second,
		},
		{
			name:     "leading newlines and emoji variation selector",
			text:     "\n\uFE0F" + noticeText("02h05m07s"),
			expected: 2*time.Hour + 5*time.Minute + 7*time.Second,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := response200JSON(jsonResponsesDocument(test.text))
			if resetDebug, matchedDebug := matchAntigravityQuotaNotice(test.text); !matchedDebug {
				t.Logf("DEBUG raw=%q", test.text)
				t.Fail()
			} else if resetDebug != test.expected {
				t.Fatalf("direct matcher reset = %v, want %v", resetDebug, test.expected)
			}
			classifyErr := classifyAntigravityQuotaNotice(response)
			notice, ok := classifyErr.(*QuotaNoticeError)
			if !ok {
				t.Fatalf("classifyAntigravityQuotaNotice() = %v, want *QuotaNoticeError", classifyErr)
			}
			if notice.ResetIn != test.expected {
				t.Fatalf("ResetIn = %v, want %v", notice.ResetIn, test.expected)
			}
			if response.Body != nil {
				t.Fatalf("upstream body left open after notice classification")
			}
		})
	}
}

func TestClassifyAntigravityJSONLegitMentionPassesThrough(t *testing.T) {
	body := jsonResponsesDocument("About quota limits: " + noticeText("54h24m58s") + " and more generated prose.")
	response := response200JSON(body)
	if classifyErr := classifyAntigravityQuotaNotice(response); classifyErr != nil {
		t.Fatalf("classifyAntigravityQuotaNotice() = %v, want nil", classifyErr)
	}
	replayed, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		t.Fatalf("replay read: %v", readErr)
	}
	if !bytes.Equal(replayed, body) {
		t.Fatalf("replayed body differs from original content")
	}
}

func TestClassifyAntigravityJSONEmptyPassesThrough(t *testing.T) {
	body := jsonResponsesDocument("")
	response := response200JSON(body)
	if classifyErr := classifyAntigravityQuotaNotice(response); classifyErr != nil {
		t.Fatalf("classifyAntigravityQuotaNotice() = %v, want nil", classifyErr)
	}
	replayed, _ := io.ReadAll(response.Body)
	if !bytes.Equal(replayed, body) {
		t.Fatalf("replayed body differs from original content")
	}
}

func TestClassifyUnknownMediaTypeNeverTouchesBody(t *testing.T) {
	response := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/xhtml+xml"}},
		Body:       io.NopCloser(bytes.NewReader([]byte("<html>" + noticeText("54h24m58s") + "</html>"))),
	}
	if classifyErr := classifyAntigravityQuotaNotice(response); classifyErr != nil {
		t.Fatalf("classifyAntigravityQuotaNotice() = %v, want nil", classifyErr)
	}
}

func TestClassifyAntigravitySSENoticeBlocks(t *testing.T) {
	payload := noticeText("54h24m58s")
	stream := response200SSE(
		"event: response.created\ndata: {\"type\":\"response.created\"}\n\n",
		sseEvent(`{"type":"response.output_text.delta","delta":`+jsonString(payload)+`}`),
		"event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"output_text\":\"\"}}\n\n",
		"data: [DONE]\n\n",
	)
	classifyErr := classifyAntigravityQuotaNotice(stream)
	notice, ok := classifyErr.(*QuotaNoticeError)
	if !ok {
		t.Fatalf("classifyAntigravityQuotaNotice() = %v, want *QuotaNoticeError", classifyErr)
	}
	if notice.ResetIn != 54*time.Hour+24*time.Minute+58*time.Second {
		t.Fatalf("ResetIn = %v", notice.ResetIn)
	}
}

func TestClassifyAntigravitySSENoticeSplitAcrossDeltasBlocks(t *testing.T) {
	notice := noticeText("54h24m58s")
	half := len(notice) / 2
	stream := response200SSE(
		sseEvent(`{"type":"response.output_text.delta","delta":`+jsonString(notice[:half])+`}`),
		sseEvent(`{"type":"response.output_text.delta","delta":`+jsonString(notice[half:])+`}`),
		"data: [DONE]\n\n",
	)
	classifyErr := classifyAntigravityQuotaNotice(stream)
	if _, ok := classifyErr.(*QuotaNoticeError); !ok {
		t.Fatalf("classifyAntigravityQuotaNotice() = %v, want *QuotaNoticeError", classifyErr)
	}
}

func TestClassifyAntigravitySSEIncompleteNoticePassesThrough(t *testing.T) {
	partial := "⚠ Individual quota reached. Please upgrade"
	stream := response200SSE(sseEvent(`{"type":"response.output_text.delta","delta":` + jsonString(partial) + `}`))
	if classifyErr := classifyAntigravityQuotaNotice(stream); classifyErr != nil {
		t.Fatalf("classifyAntigravityQuotaNotice() = %v, want nil (inconclusive prefix)", classifyErr)
	}
}

func TestClassifyAntigravitySSELegitMentionPassesThrough(t *testing.T) {
	text := "The upstream said: " + noticeText("54h24m58s") + " — quoted as evidence."
	stream := response200SSE(sseEvent(`{"type":"response.output_text.delta","delta":` + jsonString(text) + `}`))
	if classifyErr := classifyAntigravityQuotaNotice(stream); classifyErr != nil {
		t.Fatalf("classifyAntigravityQuotaNotice() = %v, want nil", classifyErr)
	}
}

func TestClassifyAntigravitySSEOnlyItemDoneNoticeBlocks(t *testing.T) {
	payload := noticeText("54h24m58s")
	stream := response200SSE(
		"event: response.created\ndata: {\"type\":\"response.created\"}\n\n",
		"event: response.output_item.done\ndata: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":"+jsonString(payload)+"}]}}\n\n",
	)
	classifyErr := classifyAntigravityQuotaNotice(stream)
	if _, ok := classifyErr.(*QuotaNoticeError); !ok {
		t.Fatalf("classifyAntigravityQuotaNotice() = %v, want *QuotaNoticeError", classifyErr)
	}
}

func TestClassifyErrorStatusNeverClassifies(t *testing.T) {
	response := response200JSON(jsonResponsesDocument(noticeText("54h24m58s")))
	response.StatusCode = http.StatusServiceUnavailable
	if classifyErr := classifyAntigravityQuotaNotice(response); classifyErr != nil {
		t.Fatalf("classifyAntigravityQuotaNotice() = %v, want nil", classifyErr)
	}
}

func TestMatchAntigravityQuotaNoticeRejectsNearMatches(t *testing.T) {
	negatives := []string{
		"",
		"⚠ Individual quota reached.",
		noticeText("54h24m58s") + " More text follows.",
		"Notes: " + noticeText("54h24m58s"),
		"⚠ Individual quota nearly reached. Please upgrade your subscription to increase your limits. Resets in 54h24m58s.",
	}
	for _, text := range negatives {
		if _, matched := matchAntigravityQuotaNotice(text); matched {
			t.Fatalf("matchAntigravityQuotaNotice(%q) matched, want no match", text)
		}
	}
}

func TestConnextoProxyOpenTargetResponsesClassifiesAntigravityNotice(t *testing.T) {
	notice := noticeText("53h59m59s")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sseEvent(`{"type":"response.output_text.delta","delta":` + jsonString(notice) + `}`)))
	}))
	defer upstream.Close()

	client, err := NewConnextoProxyClient(Settings{
		BaseURL:           upstream.URL,
		APIKey:            "test-key",
		Enabled:           true,
		LogicalProviderID: AntigravityID,
	}, upstream.Client())
	if err != nil {
		t.Fatalf("NewConnextoProxyClient: %v", err)
	}
	_, openErr := client.OpenTargetResponses(testContext(), ResponsesTarget{
		ProviderID: AntigravityID,
		ModelID:    "antigravity:claude-opus-4-6-thinking",
	}, []byte("{}"), "application/json", "")
	noticeErr, ok := openErr.(*QuotaNoticeError)
	if !ok {
		t.Fatalf("OpenTargetResponses() error = %v (type %T), want *QuotaNoticeError", openErr, openErr)
	}
	if noticeErr.ResetIn != 53*time.Hour+59*time.Minute+59*time.Second {
		t.Fatalf("ResetIn = %v", noticeErr.ResetIn)
	}
}

func TestConnextoProxyOpenTargetResponsesCursorIsNotClassified(t *testing.T) {
	notice := noticeText("53h59m59s")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(jsonResponsesDocument(notice))
	}))
	defer upstream.Close()

	client, err := NewConnextoProxyClient(Settings{
		BaseURL:           upstream.URL,
		APIKey:            "test-key",
		Enabled:           true,
		LogicalProviderID: CursorID,
	}, upstream.Client())
	if err != nil {
		t.Fatalf("NewConnextoProxyClient: %v", err)
	}
	response, openErr := client.OpenTargetResponses(testContext(), ResponsesTarget{
		ProviderID: CursorID,
		ModelID:    "composer-2.5",
	}, []byte("{}"), "application/json", "")
	if openErr != nil {
		t.Fatalf("OpenTargetResponses() error = %v, want nil", openErr)
	}
	if response == nil {
		t.Fatalf("OpenTargetResponses() response = nil, want upstream response")
	}
}
