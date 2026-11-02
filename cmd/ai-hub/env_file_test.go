package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultEnvFilePathUsesModuleRoot(t *testing.T) {
	moduleDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte("module test\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	serviceDir := filepath.Join(moduleDir, "cmd", "ai-hub")
	if err := os.MkdirAll(serviceDir, 0o700); err != nil {
		t.Fatalf("create service directory: %v", err)
	}

	want := filepath.Join(moduleDir, ".env")
	for _, startDir := range []string{moduleDir, serviceDir} {
		if got := defaultEnvFilePathFrom(startDir); got != want {
			t.Errorf("defaultEnvFilePathFrom(%q) = %q, want %q", startDir, got, want)
		}
	}
}

func TestDefaultEnvFilePathFallsBackToWorkingDirectoryOutsideModule(t *testing.T) {
	workingDir := t.TempDir()
	want := filepath.Join(workingDir, ".env")

	if got := defaultEnvFilePathFrom(workingDir); got != want {
		t.Fatalf("defaultEnvFilePathFrom(%q) = %q, want %q", workingDir, got, want)
	}
}
