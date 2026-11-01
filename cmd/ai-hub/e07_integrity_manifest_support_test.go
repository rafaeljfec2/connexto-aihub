package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	e07IntegrityManifestName    = "e07-integrity-manifest.json"
	e07IntegrityManifestPerm    = 0o600
	e07IntegrityManifestDirPerm = 0o700
)

// e07PrivateManifestDirectory selects the private manifest directory and
// rejects any location inside the repository.
func e07PrivateManifestDirectory(repositoryRoot, tempRoot string) (string, error) {
	repository, err := e07ResolveDirectory(repositoryRoot)
	if err != nil {
		return "", err
	}
	temporary, err := e07ResolveDirectory(tempRoot)
	if err != nil {
		return "", err
	}
	if e07PathInside(repository, temporary) {
		return "", errors.New("private integrity manifest directory is inside the repository")
	}

	return filepath.Join(temporary, "integrity"), nil
}

func e07PathInside(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	if relative == "." {
		return true
	}

	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

// e07SaveIntegrityManifest writes the private before-state manifest outside the
// repository with owner-only permissions.
func e07SaveIntegrityManifest(dir string, snapshot e07IntegritySnapshot) (string, error) {
	if err := os.MkdirAll(dir, e07IntegrityManifestDirPerm); err != nil {
		return "", errors.New("private integrity manifest directory could not be created")
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return "", errors.New("private integrity manifest could not be encoded")
	}
	path := filepath.Join(dir, e07IntegrityManifestName)
	if err := os.WriteFile(path, payload, e07IntegrityManifestPerm); err != nil {
		return "", errors.New("private integrity manifest could not be written")
	}
	if err := os.Chmod(path, e07IntegrityManifestPerm); err != nil {
		return "", errors.New("private integrity manifest could not be protected")
	}

	return path, nil
}

func e07LoadIntegrityManifest(path string) (e07IntegritySnapshot, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return e07IntegritySnapshot{}, errors.New("private integrity manifest could not be read")
	}
	var snapshot e07IntegritySnapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return e07IntegritySnapshot{}, errors.New("private integrity manifest was invalid")
	}
	if snapshot.Tree == "" || snapshot.Env == "" || snapshot.Tracked == "" || snapshot.GitState == "" {
		return e07IntegritySnapshot{}, errors.New("private integrity manifest was incomplete")
	}

	return snapshot, nil
}
