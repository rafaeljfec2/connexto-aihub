package gateway

import (
	"bytes"
	"testing"

	"aihub/internal/execution"
)

func TestMaskVirtualAutoModelRewritesJSONAndSpacedForms(t *testing.T) {
	in := []byte(`{"model":"composer-2.5","response":{"model": "composer-2.5"}}`)
	out, n := maskVirtualAutoModel(in, "composer-2.5", "aihub/auto")
	if n != 2 {
		t.Fatalf("rewrites=%d want 2", n)
	}
	if got := string(out); got != `{"model":"aihub/auto","response":{"model": "aihub/auto"}}` {
		t.Fatalf("out=%s", got)
	}
}

func TestMaskVirtualAutoModelIgnoresUnrelatedText(t *testing.T) {
	in := []byte(`{"output_text":"use composer-2.5 carefully"}`)
	out, n := maskVirtualAutoModel(in, "composer-2.5", "aihub/auto")
	if n != 0 || string(out) != string(in) {
		t.Fatalf("n=%d out=%s", n, out)
	}
}

type captureSink struct {
	chunks [][]byte
}

func (s *captureSink) Commit(int, string) {}
func (s *captureSink) WriteChunk(chunk []byte) error {
	s.chunks = append(s.chunks, append([]byte{}, chunk...))
	return nil
}

var _ execution.Sink = (*captureSink)(nil)

func TestVirtualAutoModelSinkMasksAcrossChunkBoundaries(t *testing.T) {
	inner := &captureSink{}
	sink := newVirtualAutoModelSink(inner, "composer-2.5")

	prefix := []byte(`data: {"type":"response.created","response":{"mod`)
	mid := []byte(`el":"composer-2.5","id":"r1"}}` + "\n\n")
	if err := sink.WriteChunk(prefix); err != nil {
		t.Fatal(err)
	}
	if err := sink.WriteChunk(mid); err != nil {
		t.Fatal(err)
	}
	sink.Commit(200, "text/event-stream")

	joined := bytes.Join(inner.chunks, nil)
	if bytes.Contains(joined, []byte(`"model":"composer-2.5"`)) {
		t.Fatalf("physical model leaked across chunks: %s", joined)
	}
	if !bytes.Contains(joined, []byte(`"model":"aihub/auto"`)) {
		t.Fatalf("virtual model missing: %s", joined)
	}
}
