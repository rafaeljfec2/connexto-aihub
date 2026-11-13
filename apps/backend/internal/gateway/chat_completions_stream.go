package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// chatStreamTranslator rewrites Responses SSE into OpenAI Chat Completions chunks.
type chatStreamTranslator struct {
	inner       http.ResponseWriter
	model       string
	id          string
	created     int64
	status      int
	passthrough bool
	headerSent  bool
	finished    bool
	buf         []byte
	roleSent      bool
	toolIndex     int
	seenCallIDs   map[string]struct{}
	pendingTools  map[string]chatStreamToolCall
}

func newChatStreamTranslator(w http.ResponseWriter, model, id string) *chatStreamTranslator {
	return &chatStreamTranslator{
		inner:        w,
		model:        model,
		id:           id,
		created:      time.Now().Unix(),
		status:       http.StatusOK,
		seenCallIDs:  make(map[string]struct{}),
		pendingTools: make(map[string]chatStreamToolCall),
	}
}

func (t *chatStreamTranslator) Header() http.Header {
	return t.inner.Header()
}

func (t *chatStreamTranslator) WriteHeader(statusCode int) {
	t.status = statusCode
	if statusCode != http.StatusOK {
		t.passthrough = true
		t.inner.WriteHeader(statusCode)
		t.headerSent = true
	}
}

func (t *chatStreamTranslator) Flush() {
	if flusher, ok := t.inner.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (t *chatStreamTranslator) Write(p []byte) (int, error) {
	if t.passthrough {
		return t.inner.Write(p)
	}
	t.buf = append(t.buf, p...)
	for {
		idx := bytes.Index(t.buf, []byte("\n\n"))
		if idx < 0 {
			return len(p), nil
		}
		block := t.buf[:idx]
		t.buf = t.buf[idx+2:]
		if err := t.emitTranslatedBlock(block); err != nil {
			return 0, err
		}
	}
}

func (t *chatStreamTranslator) flushFinish() error {
	if t.passthrough || t.finished {
		return nil
	}
	t.finished = true
	finishReason := "stop"
	if t.toolIndex > 0 {
		finishReason = "tool_calls"
	}
	if err := t.writeChatChunk(nil, finishReason); err != nil {
		return err
	}
	_, err := fmt.Fprintf(t.inner, "data: [DONE]\n\n")
	return err
}

func (t *chatStreamTranslator) ensureHeader() error {
	if t.headerSent {
		return nil
	}
	t.inner.Header().Set("Content-Type", "text/event-stream")
	t.inner.Header().Set("Cache-Control", "no-cache")
	t.inner.WriteHeader(http.StatusOK)
	t.headerSent = true
	if flusher, ok := t.inner.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func (t *chatStreamTranslator) emitTranslatedBlock(block []byte) error {
	eventName, data := parseSSEBlock(block)
	if len(data) == 0 || bytes.Equal(data, []byte("[DONE]")) {
		return nil
	}
	if isResponsesStreamError(eventName, data) {
		t.buf = nil
		if t.headerSent {
			// Chat wire already started — never mix Responses SSE into the stream.
			return t.emitChatStreamError(streamErrorMessage(data))
		}
		t.passthrough = true
		if err := t.ensureHeader(); err != nil {
			return err
		}
		_, err := t.inner.Write(append(block, '\n', '\n'))
		return err
	}
	if itemID, partial, argsOnly, partialOK := responsesStreamToolCallPartial(eventName, data); partialOK {
		if argsOnly {
			base := t.pendingTools[itemID]
			merged := mergePartialToolCall(base, partial)
			t.pendingTools[itemID] = merged
			if toolCallReady(merged) {
				if err := t.emitToolCall(merged); err != nil {
					return err
				}
				delete(t.pendingTools, itemID)
			}
		} else if partial.arguments == "" {
			t.pendingTools[itemID] = partial
		} else if err := t.emitToolCall(partial); err != nil {
			return err
		}
	} else if toolCall, ok := responsesStreamToolCall(eventName, data); ok {
		if err := t.emitToolCall(toolCall); err != nil {
			return err
		}
	}

	delta := extractResponsesStreamDelta(eventName, data)
	if delta == "" && !isResponsesTerminalEvent(eventName, data) {
		return nil
	}

	if err := t.ensureHeader(); err != nil {
		return err
	}

	if !t.roleSent {
		if err := t.writeChatChunk(map[string]any{"role": "assistant"}, ""); err != nil {
			return err
		}
		t.roleSent = true
	}

	if delta != "" {
		if err := t.writeChatChunk(map[string]any{"content": delta}, ""); err != nil {
			return err
		}
	}

	if isResponsesTerminalEvent(eventName, data) {
		for _, toolCall := range responsesCompletedOutputTools(data) {
			if err := t.emitToolCall(toolCall); err != nil {
				return err
			}
		}
		return t.flushFinish()
	}
	return nil
}

func (t *chatStreamTranslator) emitToolCall(toolCall chatStreamToolCall) error {
	if toolCall.callID == "" || toolCall.name == "" {
		return nil
	}
	if _, seen := t.seenCallIDs[toolCall.callID]; seen {
		return nil
	}
	t.seenCallIDs[toolCall.callID] = struct{}{}
	if err := t.ensureHeader(); err != nil {
		return err
	}
	if !t.roleSent {
		if err := t.writeChatChunk(map[string]any{"role": "assistant"}, ""); err != nil {
			return err
		}
		t.roleSent = true
	}
	callType := "function"
	if toolCall.custom {
		callType = "custom"
	}
	args := toolCall.arguments
	if args == "" {
		args = "{}"
	}
	return t.writeChatToolCallChunk(map[string]any{
		"index": t.toolIndex, "id": toolCall.callID, "type": callType,
		"function": map[string]any{"name": toolCall.name, "arguments": args},
	})
}

func (t *chatStreamTranslator) writeChatChunk(delta map[string]any, finishReason string) error {
	deltaField := any(map[string]any{})
	if delta != nil {
		deltaField = delta
	}
	finish := any(nil)
	if finishReason != "" {
		finish = finishReason
	}
	choice := map[string]any{
		"index":         0,
		"delta":         deltaField,
		"finish_reason": finish,
	}
	payload := map[string]any{
		"id":                 t.id,
		"object":             "chat.completion.chunk",
		"created":            t.created,
		"model":              t.model,
		"system_fingerprint": nil,
		"choices":            []map[string]any{choice},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(t.inner, "data: %s\n\n", encoded); err != nil {
		return err
	}
	if flusher, ok := t.inner.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func (t *chatStreamTranslator) emitChatStreamError(message string) error {
	if t.finished {
		return nil
	}
	t.finished = true
	t.passthrough = true
	if err := t.ensureHeader(); err != nil {
		return err
	}
	if message == "" {
		message = "the upstream stream failed"
	}
	payload, err := json.Marshal(map[string]any{
		"error": map[string]any{
			"message": message,
			"type":    "server_error",
			"code":    nil,
		},
	})
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(t.inner, "data: %s\n\n", payload); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(t.inner, "data: [DONE]\n\n"); err != nil {
		return err
	}
	if flusher, ok := t.inner.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func streamErrorMessage(data []byte) string {
	var envelope struct {
		Message string `json:"message"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return ""
	}
	if envelope.Error.Message != "" {
		return envelope.Error.Message
	}
	return envelope.Message
}

func (t *chatStreamTranslator) writeChatToolCallChunk(toolCall map[string]any) error {
	payload := map[string]any{
		"id": t.id, "object": "chat.completion.chunk", "created": t.created,
		"model": t.model, "system_fingerprint": nil,
		"choices": []map[string]any{{
			"index":         0,
			"delta":         map[string]any{"tool_calls": []any{toolCall}},
			"finish_reason": nil,
		}},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(t.inner, "data: %s\n\n", encoded); err != nil {
		return err
	}
	if flusher, ok := t.inner.(http.Flusher); ok {
		flusher.Flush()
	}
	t.toolIndex++
	return nil
}

func extractResponsesStreamDelta(eventName string, data []byte) string {
	var envelope struct {
		Type  string `json:"type"`
		Delta string `json:"delta"`
	}
	if json.Unmarshal(data, &envelope) == nil {
		if envelope.Delta != "" {
			return envelope.Delta
		}
		if strings.Contains(envelope.Type, "output_text.delta") {
			return envelope.Delta
		}
	}
	if strings.Contains(eventName, "output_text.delta") {
		return envelope.Delta
	}
	var generic map[string]json.RawMessage
	if json.Unmarshal(data, &generic) != nil {
		return ""
	}
	if raw, ok := generic["delta"]; ok {
		if text, ok := jsonRawString(raw); ok {
			return text
		}
	}
	return ""
}

func isResponsesTerminalEvent(eventName string, data []byte) bool {
	if strings.Contains(eventName, "response.completed") || strings.Contains(eventName, "response.done") {
		return true
	}
	return bytes.Contains(data, []byte(`"type":"response.completed"`)) ||
		bytes.Contains(data, []byte(`"type":"response.done"`))
}

func isResponsesStreamError(eventName string, data []byte) bool {
	if strings.Contains(eventName, "error") {
		return true
	}
	var envelope struct {
		Type  string          `json:"type"`
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return bytes.Contains(data, []byte(`"type":"error"`))
	}
	if envelope.Type == "error" || strings.Contains(envelope.Type, ".error") {
		return true
	}
	if len(envelope.Error) == 0 || string(envelope.Error) == "null" {
		return false
	}
	return true
}

func parseSSEBlock(block []byte) (eventName string, data []byte) {
	lines := bytes.Split(block, []byte("\n"))
	for _, line := range lines {
		if bytes.HasPrefix(line, []byte("event:")) {
			eventName = strings.TrimSpace(string(bytes.TrimPrefix(line, []byte("event:"))))
		}
		if bytes.HasPrefix(line, []byte("data:")) {
			data = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		}
	}
	return eventName, data
}
