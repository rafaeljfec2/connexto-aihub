package gateway

import (
	"bytes"

	"aihub/internal/execution"
	"aihub/internal/routing"
)

// virtualAutoModelSink rewrites upstream response model fields back to the
// virtual Auto id so clients (Codex) do not treat a physical pool pick as a
// mid-thread model change. Selection remains free AW-RR server-side.
//
// WriteChunk may hold a trailing partial `"model":"..."` match across chunk
// boundaries so SSE splits cannot leak the physical model id.
type virtualAutoModelSink struct {
	inner    execution.Sink
	physical string
	virtual  string
	hold     []byte
}

func newVirtualAutoModelSink(inner execution.Sink, physicalModelID string) *virtualAutoModelSink {
	return &virtualAutoModelSink{
		inner:    inner,
		physical: physicalModelID,
		virtual:  routing.VirtualModelAuto,
	}
}

func (s *virtualAutoModelSink) Commit(status int, contentType string) {
	if len(s.hold) > 0 {
		_ = s.inner.WriteChunk(s.hold)
		s.hold = nil
	}
	s.inner.Commit(status, contentType)
}

func (s *virtualAutoModelSink) WriteChunk(chunk []byte) error {
	if len(chunk) == 0 && len(s.hold) == 0 {
		return nil
	}
	data := chunk
	if len(s.hold) > 0 {
		data = append(append([]byte{}, s.hold...), chunk...)
		s.hold = nil
	}
	masked, _ := maskVirtualAutoModel(data, s.physical, s.virtual)
	emit, hold := splitTrailingPartialModelKey(masked, s.physical)
	if len(hold) > 0 {
		s.hold = hold
	}
	if len(emit) == 0 {
		return nil
	}
	return s.inner.WriteChunk(emit)
}

func maskVirtualAutoModel(chunk []byte, physical, virtual string) ([]byte, int) {
	if physical == "" || virtual == "" || physical == virtual || !bytes.Contains(chunk, []byte(physical)) {
		return chunk, 0
	}
	patterns := []struct {
		from []byte
		to   []byte
	}{
		{[]byte(`"model":"` + physical + `"`), []byte(`"model":"` + virtual + `"`)},
		{[]byte(`"model": "` + physical + `"`), []byte(`"model": "` + virtual + `"`)},
	}
	out := chunk
	total := 0
	for _, p := range patterns {
		if !bytes.Contains(out, p.from) {
			continue
		}
		count := bytes.Count(out, p.from)
		out = bytes.ReplaceAll(out, p.from, p.to)
		total += count
	}
	return out, total
}

func splitTrailingPartialModelKey(buf []byte, physical string) (emit []byte, hold []byte) {
	if physical == "" || len(buf) == 0 {
		return buf, nil
	}
	patterns := [][]byte{
		[]byte(`"model":"` + physical + `"`),
		[]byte(`"model": "` + physical + `"`),
	}
	maxHold := 0
	for _, p := range patterns {
		if len(p)-1 > maxHold {
			maxHold = len(p) - 1
		}
	}
	if maxHold <= 0 || len(buf) < maxHold {
		// Still check shorter prefixes over the whole buffer.
		maxHold = len(buf)
	}
	start := len(buf) - maxHold
	if start < 0 {
		start = 0
	}
	for i := start; i < len(buf); i++ {
		suffix := buf[i:]
		for _, p := range patterns {
			if len(suffix) < len(p) && bytes.HasPrefix(p, suffix) {
				return buf[:i], append([]byte{}, suffix...)
			}
		}
	}
	return buf, nil
}
