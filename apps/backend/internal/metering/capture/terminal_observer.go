package capture

import "bytes"

// SSETerminalObserver tracks whether a forwarded SSE byte stream contains a
// terminal Responses event (response.completed / response.done), using framed
// parsing instead of raw substring search over the whole chunk.
type SSETerminalObserver struct {
	hold        []byte
	SawTerminal bool
}

// ObserveChunk ingests another forwarded chunk from the upstream stream.
func (o *SSETerminalObserver) ObserveChunk(chunk []byte) {
	if o == nil || o.SawTerminal || len(chunk) == 0 {
		return
	}
	combined := append(o.hold, chunk...)
	for {
		idx := bytes.Index(combined, []byte("\n\n"))
		if idx < 0 {
			o.hold = combined
			return
		}
		block := bytes.TrimSpace(combined[:idx])
		combined = combined[idx+2:]
		if len(block) == 0 {
			continue
		}
		eventName, data := parseSSEBlock(block)
		if terminalUsageEvent(eventName, data) {
			o.SawTerminal = true
			o.hold = nil
			return
		}
	}
}
