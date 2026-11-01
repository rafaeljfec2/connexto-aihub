//go:build !linux

package main

import "testing"

func e07CodexProcessActive() (bool, error) {
	return e07CodexProcessUnsupported()
}

func TestE07CodexProcessActiveFailsClosedOnUnsupportedPlatform(t *testing.T) {
	active, err := e07CodexProcessActive()
	if err == nil || active {
		t.Fatal("unsupported Codex process detection did not fail closed")
	}
}
