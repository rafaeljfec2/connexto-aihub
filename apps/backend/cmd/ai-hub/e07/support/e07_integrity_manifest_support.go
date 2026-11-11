package support

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	E07IntegrityManifestName    = "e07-integrity-manifest.json"
	E07IntegrityManifestPerm    = 0o600
	E07IntegrityManifestDirPerm = 0o700
)

// e07PrivateManifestDirectory selects the private manifest directory and
// rejects any location inside the repository.
func E07PrivateManifestDirectory(repositoryRoot, tempRoot string) (string, error) {
	repository, err := E07ResolveDirectory(repositoryRoot)
	if err != nil {
		return "", err
	}
	temporary, err := E07ResolveDirectory(tempRoot)
	if err != nil {
		return "", err
	}
	if E07PathInside(repository, temporary) {
		return "", errors.New("private integrity manifest directory is inside the repository")
	}

	return filepath.Join(temporary, "integrity"), nil
}

func E07PathInside(parent, child string) bool {
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
func E07SaveIntegrityManifest(dir string, snapshot E07IntegritySnapshot) (string, error) {
	if err := os.MkdirAll(dir, E07IntegrityManifestDirPerm); err != nil {
		return "", errors.New("private integrity manifest directory could not be created")
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return "", errors.New("private integrity manifest could not be encoded")
	}
	path := filepath.Join(dir, E07IntegrityManifestName)
	if err := os.WriteFile(path, payload, E07IntegrityManifestPerm); err != nil {
		return "", errors.New("private integrity manifest could not be written")
	}
	if err := os.Chmod(path, E07IntegrityManifestPerm); err != nil {
		return "", errors.New("private integrity manifest could not be protected")
	}

	return path, nil
}

func E07LoadIntegrityManifest(path string) (E07IntegritySnapshot, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return E07IntegritySnapshot{}, errors.New("private integrity manifest could not be read")
	}
	var snapshot E07IntegritySnapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return E07IntegritySnapshot{}, errors.New("private integrity manifest was invalid")
	}
	if snapshot.Tree == "" || snapshot.Env == "" || snapshot.Tracked == "" || snapshot.GitState == "" {
		return E07IntegritySnapshot{}, errors.New("private integrity manifest was incomplete")
	}

	return snapshot, nil
}
