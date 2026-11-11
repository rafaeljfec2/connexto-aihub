package support

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func E07SetCodexSnapshotHome(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	t.Setenv("CODEX_SQLITE_HOME", "")
}

func E07SaveCodexTreeManifest(directory, phase string, entries []E07CodexTreeEntry) (string, error) {
	payload, err := json.Marshal(entries)
	if err != nil {
		return "", errors.New("private Codex metadata manifest could not be encoded")
	}
	path := filepath.Join(directory, "codex-tree-"+phase+".manifest")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", errors.New("private Codex metadata manifest could not be created")
	}
	if _, err := file.Write(payload); err != nil {
		_ = file.Close()
		return "", errors.New("private Codex metadata manifest could not be written")
	}
	if err := file.Close(); err != nil {
		return "", errors.New("private Codex metadata manifest could not be closed")
	}
	return path, nil
}

func E07LoadCodexTreeManifest(path string) ([]E07CodexTreeEntry, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("private Codex metadata manifest could not be read")
	}
	var entries []E07CodexTreeEntry
	if err := json.Unmarshal(payload, &entries); err != nil {
		return nil, errors.New("private Codex metadata manifest could not be decoded")
	}
	return entries, nil
}
