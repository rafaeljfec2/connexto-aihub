package gateway

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
)

// NewChatCompletionsHandler adapts OpenAI Chat Completions to the hub Responses pipeline.
func NewChatCompletionsHandler(responses http.Handler) http.Handler {
	if responses == nil {
		panic("chat completions handler requires a responses handler")
	}
	return &chatCompletionsHandler{responses: responses}
}

type chatCompletionsHandler struct {
	responses http.Handler
}

func (h *chatCompletionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, invalidRequestType, "only POST is supported")
		return
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, invalidRequestType, "the request body could not be read")
		return
	}

	responsesPayload, clientModel, stream, err := chatRequestToResponses(payload)
	if err != nil {
		switch {
		case errors.Is(err, errChatModelRequired):
			writeJSONError(w, http.StatusBadRequest, invalidRequestType, "model is required")
		case errors.Is(err, errChatMessagesRequired):
			writeJSONError(w, http.StatusBadRequest, invalidRequestType, "messages is required")
		default:
			writeJSONError(w, http.StatusBadRequest, invalidRequestType, "the request body is invalid")
		}
		return
	}

	proxy := r.Clone(r.Context())
	proxy.Method = http.MethodPost
	proxy.URL.Path = "/v1/responses"
	proxy.Body = io.NopCloser(bytes.NewReader(responsesPayload))
	proxy.ContentLength = int64(len(responsesPayload))
	if proxy.Header.Get("Content-Type") == "" {
		proxy.Header.Set("Content-Type", "application/json")
	}

	if stream {
		chatID, idErr := newChatCompletionID()
		if idErr != nil {
			writeJSONError(w, http.StatusInternalServerError, executionFailedType, executionFailedMsg)
			return
		}
		translator := newChatStreamTranslator(w, clientModel, chatID)
		h.responses.ServeHTTP(translator, proxy)
		if len(translator.buf) > 0 {
			_ = translator.emitTranslatedBlock(translator.buf)
			translator.buf = nil
		}
		if !translator.passthrough && translator.headerSent {
			_ = translator.flushFinish()
		}
		return
	}

	recorder := &responseRecorder{}
	h.responses.ServeHTTP(recorder, proxy)

	for key, values := range recorder.Header() {
		if strings.EqualFold(key, "Content-Length") {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	if recorder.status != 0 && recorder.status != http.StatusOK {
		w.WriteHeader(recorder.status)
		_, _ = w.Write(recorder.body.Bytes())
		return
	}

	chatBody, err := responsesBodyToChatCompletion(recorder.body.Bytes(), clientModel)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, executionFailedType, "the chat completion response could not be converted")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(chatBody)
}

type responseRecorder struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (r *responseRecorder) Header() http.Header {
	if r.header == nil {
		r.header = make(http.Header)
	}
	return r.header
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
}

func (r *responseRecorder) Write(p []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(p)
}

func newChatCompletionID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "chatcmpl_" + hex.EncodeToString(buf), nil
}
