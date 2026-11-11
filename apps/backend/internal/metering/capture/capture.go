package capture

import (
	"bytes"
	"encoding/json"
	"strings"
)

const maxTailBytes = 64 << 10

// Result holds the final provider usage payload discovered while forwarding bytes.
type Result struct {
	// FinalEventData is the raw JSON object from the terminal SSE data line, when present.
	FinalEventData []byte
	// ResponseBodyTail holds a bounded suffix for non-SSE JSON responses.
	ResponseBodyTail []byte
	Streamed         bool
}

// StreamTap observes forwarded chunks without mutating them.
type StreamTap struct {
	buf bytes.Buffer
}

// NewStreamTap builds a tap for one execution response stream.
func NewStreamTap() *StreamTap {
	return &StreamTap{}
}

// Observe records another forwarded chunk.
func (t *StreamTap) Observe(chunk []byte) {
	if t == nil || len(chunk) == 0 {
		return
	}
	_, _ = t.buf.Write(chunk)
	if t.buf.Len() > maxTailBytes {
		overflow := t.buf.Len() - maxTailBytes
		remaining := t.buf.Bytes()[overflow:]
		t.buf.Reset()
		_, _ = t.buf.Write(remaining)
	}
}

// Finalize parses the buffered stream and returns capture metadata.
func (t *StreamTap) Finalize(contentType string) Result {
	if t == nil {
		return Result{}
	}
	body := t.buf.Bytes()
	if len(body) == 0 {
		return Result{}
	}
	if isEventStream(contentType) || looksLikeSSE(body) {
		if data, ok := finalSSEUsageData(body); ok {
			return Result{FinalEventData: data, Streamed: true}
		}
		return Result{Streamed: true}
	}
	tail := append([]byte(nil), body...)
	if len(tail) > maxTailBytes {
		tail = tail[len(tail)-maxTailBytes:]
	}
	return Result{ResponseBodyTail: tail, Streamed: false}
}

func isEventStream(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/event-stream")
}

func looksLikeSSE(body []byte) bool {
	return bytes.Contains(body, []byte("data:")) && bytes.Contains(body, []byte("\n\n"))
}

func finalSSEUsageData(body []byte) ([]byte, bool) {
	var last []byte
	for _, block := range splitSSEBlocks(body) {
		eventName, data := parseSSEBlock(block)
		if len(data) == 0 || bytes.Equal(data, []byte("[DONE]")) {
			continue
		}
		if !terminalUsageEvent(eventName, data) {
			continue
		}
		last = append([]byte(nil), data...)
	}
	if len(last) == 0 {
		return nil, false
	}
	return last, true
}

func splitSSEBlocks(body []byte) [][]byte {
	parts := bytes.Split(body, []byte("\n\n"))
	out := make([][]byte, 0, len(parts))
	for _, part := range parts {
		trimmed := bytes.TrimSpace(part)
		if len(trimmed) == 0 {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func parseSSEBlock(block []byte) (eventName string, data []byte) {
	for _, line := range bytes.Split(block, []byte("\n")) {
		if bytes.HasPrefix(line, []byte("event:")) {
			eventName = strings.TrimSpace(string(bytes.TrimPrefix(line, []byte("event:"))))
			continue
		}
		if bytes.HasPrefix(line, []byte("data:")) {
			data = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		}
	}
	return eventName, data
}

func terminalUsageEvent(eventName string, data []byte) bool {
	eventName = strings.TrimSpace(eventName)
	if eventName == "response.completed" {
		return true
	}
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return false
	}
	switch envelope.Type {
	case "response.completed", "response.done":
		return true
	default:
		return false
	}
}
