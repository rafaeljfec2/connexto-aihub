//go:build !linux

package codex

import (
	e07support "aihub/cmd/ai-hub/e07/support"
	"testing"
)

func TestE07CodexProcessActiveFailsClosedOnUnsupportedPlatform(t *testing.T) {
	active, err := e07support.E07CodexProcessActive()
	if err == nil || active {
		t.Fatal("unsupported Codex process detection did not fail closed")
	}
}
