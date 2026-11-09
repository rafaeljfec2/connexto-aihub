package providers

// Antigravity upstream availability classification.
//
// The Connexto Proxy antigravity upstream serves quota and subscription
// notices as HTTP 200 generation payloads (both JSON and SSE), which callers
// cannot distinguish from real model output at transport level. This file
// classifies those notices and turns them into a typed opening error before
// any byte is committed downstream.
//
// Detection is deliberately strict and bounded: the notice must be the
// complete model output (non-streaming) or the complete text payload carried
// by the observed stream prefix (streaming). Legitimate output that merely
// mentions quota limits is never intercepted, because only an anchored,
// whole-message match against the canonical upstream notice decides the
// classification; any divergence from the notice lead-in opts the response
// out immediately. The upstream notice text is never echoed back to the
// caller; only a sanitized structured error is produced.

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// QuotaNoticeError represents an upstream-reported quota or subscription
// exhaustion that prevented any model generation. It mirrors the
// ResponseHeaderTimeoutError classification seam: the execution package
// detects it through an unexported interface, no package cycle is created.
type QuotaNoticeError struct {
	// ResetIn is the reset window parsed from the upstream notice.
	// Zero when the notice did not announce one.
	ResetIn time.Duration
}

func (e *QuotaNoticeError) Error() string {
	return "upstream quota or subscription limit reached; no generation was produced"
}

// IsUpstreamQuotaNotice marks the typed classification consumed by execution.
func (e *QuotaNoticeError) IsUpstreamQuotaNotice() bool { return true }

// QuotaResetIn exposes the parsed reset window.
func (e *QuotaNoticeError) QuotaResetIn() time.Duration { return e.ResetIn }

// antigravityQuotaNoticeCanonicalPrefix is the fixed lead-in of the upstream
// notice. Detection requires a full anchored match; divergence from this
// prefix at any position immediately opts the response out of classification.
const antigravityQuotaNoticeCanonicalPrefix = "⚠ Individual quota reached. Please upgrade your subscription to increase your limits. Resets in "

var antigravityQuotaNoticeRE = regexp.MustCompile(
	`^⚠\s*Individual quota reached\. Please upgrade your subscription to increase your limits\. Resets in ` +
		`(\d+)h(\d+)m(\d+)s\.?$`)

// antigravityMaximumNoticeRunes bounds how much text is worth holding before
// declaring the response legitimate. The notice is ~120 runes.
const antigravityMaximumNoticeRunes = 256

// antigravityInspectBodyBytes bounds buffered inspection of a non-streaming
// body. Anything beyond it stop being inspected and is served verbatim.
const antigravityInspectBodyBytes = 8 << 20

// antigravityInspectStreamBytes bounds the streaming peek before the response
// is treated as legitimate and the relay receives the replayed bytes.
const antigravityInspectStreamBytes = 64 << 10

var antigravityWhitespaceReplacer = strings.NewReplacer("\u00A0", " ", "\u202F", " ", "\u2007", " ")

// matchAntigravityQuotaNotice reports whether text is exactly the upstream
// notice and, when it is, the parsed reset window.
func matchAntigravityQuotaNotice(text string) (time.Duration, bool) {
	normalized := normalizeAntigravityNoticeText(text)
	matches := antigravityQuotaNoticeRE.FindStringSubmatch(normalized)
	if matches == nil {
		return 0, false
	}
	var reset time.Duration
	if hours, err := strconv.Atoi(matches[1]); err == nil {
		reset += time.Duration(hours) * time.Hour
	}
	if minutes, err := strconv.Atoi(matches[2]); err == nil {
		reset += time.Duration(minutes) * time.Minute
	}
	if seconds, err := strconv.Atoi(matches[3]); err == nil {
		reset += time.Duration(seconds) * time.Second
	}
	return reset, true
}

func normalizeAntigravityNoticeText(text string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(text, "\uFE0F", ""))
	if trimmed == "" {
		return ""
	}
	return antigravityWhitespaceReplacer.Replace(trimmed)
}

// antigravityNoticeDiverged reports whether normalized text deviates from the
// canonical notice lead-in at any consumed position.
func antigravityNoticeDiverged(text string) bool {
	normalized := normalizeAntigravityNoticeText(text)
	prefix := antigravityQuotaNoticeCanonicalPrefix
	for position := 0; position < len(normalized) && position < len(prefix); position++ {
		if normalized[position] != prefix[position] {
			return true
		}
	}
	return false
}

// readUpTo reads at most limit bytes. capReached reports whether the limit
// was hit (more bytes may remain unread). A non-EOF read error keeps the
// bytes consumed so far for reassembly.
func readUpTo(body io.Reader, limit int) (buffer []byte, readErr error, capReached bool) {
	chunk := make([]byte, 16<<10)
	for len(buffer) <= limit-1 {
		read, err := body.Read(chunk)
		if read > 0 {
			budget := limit - len(buffer)
			if read > budget {
				read = budget
			}
			buffer = append(buffer, chunk[:read]...)
		}
		if err != nil {
			if err == io.EOF {
				return buffer, nil, len(buffer) >= limit
			}
			return buffer, err, len(buffer) >= limit
		}
		if read == 0 {
			return buffer, io.ErrNoProgress, len(buffer) >= limit
		}
	}
	return buffer, nil, true
}

// classifyAntigravityQuotaNotice inspects a successful upstream response and
// replaces `response.Body` with a reader that replays the buffered bytes in
// front of the remaining stream (content stays byte-for-byte). It returns a
// *QuotaNoticeError (closing the upstream body) when the payload is exactly
// the quota notice. Untouched responses (errors, unsupported content types)
// return nil without side effects.
func classifyAntigravityQuotaNotice(response *http.Response) error {
	if response == nil || response.StatusCode != http.StatusOK || response.Body == nil {
		return nil
	}
	contentType := strings.ToLower(response.Header.Get("Content-Type"))
	if contentType == "" {
		return nil
	}
	switch {
	case strings.Contains(contentType, "json"):
		return classifyAntigravityJSON(response)
	case strings.Contains(contentType, "text/event-stream"):
		return classifyAntigravitySSE(response)
	default:
		// Unknown media types are never interpreted: passthrough untouched.
		return nil
	}
}

func classifyAntigravityJSON(response *http.Response) error {
	buffer, readErr, _ := readUpTo(response.Body, antigravityInspectBodyBytes)
	if readErr != nil && len(buffer) == 0 {
		// Nothing was buffered: keep the original body untouched.
		return nil
	}
	if reset, matched := classifyAntigravityOutputText(buffer); matched {
		_ = response.Body.Close()
		response.Body = nil
		return &QuotaNoticeError{ResetIn: reset}
	}
	response.Body = io.NopCloser(io.MultiReader(bytes.NewReader(buffer), response.Body))
	return nil
}

func classifyAntigravitySSE(response *http.Response) error {
	buffer, readErr, _ := readUpTo(response.Body, antigravityInspectStreamBytes)
	if readErr != nil && len(buffer) == 0 {
		return nil
	}
	noticeErr, _ := classifyAntigravitySSEPrefix(buffer)
	if noticeErr != nil {
		_ = response.Body.Close()
		response.Body = nil
		return noticeErr
	}
	response.Body = io.NopCloser(io.MultiReader(bytes.NewReader(buffer), response.Body))
	return nil
}

// classifyAntigravityOutputText runs the whole-notice decision over a
// Responses JSON document.
func classifyAntigravityOutputText(buffer []byte) (time.Duration, bool) {
	var document struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.Unmarshal(buffer, &document); err != nil {
		return 0, false
	}
	text := document.OutputText
	if strings.TrimSpace(text) == "" {
		var parts []string
		for _, item := range document.Output {
			for _, content := range item.Content {
				if strings.TrimSpace(content.Text) != "" {
					parts = append(parts, content.Text)
				}
			}
		}
		text = strings.Join(parts, "")
	}
	if strings.TrimSpace(text) == "" {
		return 0, false
	}
	return matchAntigravityQuotaNotice(text)
}

// antigravitySSEPrefixClassifier accumulates the observed generation text of a
// stream prefix and decides blocked/passthrough. Complete blocks are fed in
// order; partial event frames are kept for later rounds.
type antigravitySSEPrefixClassifier struct {
	pending    []byte
	acc        strings.Builder
	decided    bool
	terminated bool
}

func (c *antigravitySSEPrefixClassifier) feed(raw []byte) error {
	c.pending = append(c.pending, raw...)
	for {
		index := bytes.Index(c.pending, []byte("\n\n"))
		if index < 0 {
			break
		}
		block := c.pending[:index]
		c.pending = c.pending[index+2:]
		c.feedBlock(block)
		if noticeErr := c.evaluate(); noticeErr != nil || c.decided {
			return noticeErr
		}
	}
	return nil
}

func (c *antigravitySSEPrefixClassifier) feedBlock(block []byte) {
	const (
		sseKindDelta     = "response.output_text.delta"
		sseKindCompleted = "response.completed"
		sseKindDone      = "response.done"
		sseKindItemDone  = "response.output_item.done"
	)
	for _, line := range bytes.Split(block, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		data := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if len(data) == 0 {
			continue
		}
		if bytes.Equal(data, []byte("[DONE]")) {
			c.terminated = true
			return
		}
		var event struct {
			Type     string `json:"type"`
			Delta    string `json:"delta"`
			Response struct {
				OutputText string `json:"output_text"`
			} `json:"response"`
			Item struct {
				Type    string `json:"type"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"item"`
		}
		if json.Unmarshal(data, &event) != nil {
			continue
		}
		switch event.Type {
		case sseKindDelta:
			c.acc.WriteString(event.Delta)
		case sseKindCompleted, sseKindDone:
			if c.acc.Len() == 0 {
				c.acc.WriteString(event.Response.OutputText)
			}
			c.terminated = true
			return
		case sseKindItemDone:
			if c.acc.Len() == 0 && strings.EqualFold(event.Item.Type, "message") {
				for _, content := range event.Item.Content {
					if content.Type == "output_text" || content.Type == "text" {
						c.acc.WriteString(content.Text)
					}
				}
			}
			c.terminated = true
			return
		}
	}
}

// evaluate runs the decision: a full anchored match blocks; divergence from
// the canonical lead-in, an over-long accumulation, or stream termination
// without a full match, all decide passthrough.
func (c *antigravitySSEPrefixClassifier) evaluate() error {
	text := c.acc.String()
	if reset, matched := matchAntigravityQuotaNotice(text); matched {
		c.decided = true
		return &QuotaNoticeError{ResetIn: reset}
	}
	trimmed := normalizeAntigravityNoticeText(text)
	if c.terminated || antigravityNoticeDiverged(text) || len([]rune(trimmed)) > antigravityMaximumNoticeRunes {
		c.decided = true
		return nil
	}
	return nil
}

// classifyAntigravitySSEPrefix classifies a buffered stream prefix. It
// returns (nil, false) when the prefix is inconclusive (caller treats the
// response as legitimate and replays it), (nil, true) when decided healthy,
// and (noticeErr, true) when the observed generation is exactly the notice.
func classifyAntigravitySSEPrefix(buffer []byte) (error, bool) {
	classifier := &antigravitySSEPrefixClassifier{}
	if noticeErr := classifier.feed(buffer); noticeErr != nil {
		return noticeErr, true
	}
	if !classifier.decided {
		return nil, false
	}
	return nil, true
}
