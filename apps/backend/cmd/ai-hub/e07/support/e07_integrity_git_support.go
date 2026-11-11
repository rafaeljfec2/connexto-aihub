package support

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// e07FingerprintTrackedFiles hashes the content of every Git-tracked file,
// including files that were already modified before the validation started.
func E07FingerprintTrackedFiles(root string) (string, error) {
	files, err := E07GitTrackedFiles(root)
	if err != nil {
		return "", err
	}
	hasher := sha256.New()
	_, _ = fmt.Fprintf(hasher, "tracked-files:%d\n", len(files))
	for _, relative := range files {
		path := filepath.Join(root, filepath.FromSlash(relative))
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			_, _ = fmt.Fprintf(hasher, "missing:%s\n", relative)
			continue
		}
		if err != nil {
			return "", errors.New("tracked-file fingerprint could not inspect an entry")
		}
		if err := E07DigestPath(hasher, path, relative, info, make(map[string]struct{}), new(int)); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func E07GitTrackedFiles(root string) ([]string, error) {
	command := exec.Command("git", "-C", root, "ls-files", "-z")
	output, err := command.Output()
	if err != nil {
		return nil, errors.New("Git tracked files could not be listed")
	}
	files := make([]string, 0, 128)
	for _, entry := range strings.Split(string(output), "\x00") {
		if entry != "" {
			files = append(files, entry)
		}
	}
	sort.Strings(files)
	return files, nil
}

func E07GitState(root string) (string, error) {
	secretFiles, err := E07EnvFiles(root)
	if err != nil {
		return "", err
	}
	args := []string{"-C", root, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--", "."}
	for _, relative := range secretFiles {
		args = append(args, ":(exclude,literal)"+relative)
	}
	command := exec.Command("git", args...)
	state, err := command.Output()
	if err != nil {
		return "", errors.New("Git worktree state could not be captured")
	}
	digest := sha256.Sum256(state)
	return hex.EncodeToString(digest[:]), nil
}

func E07GitRevision(root string) (string, error) {
	command := exec.Command("git", "-C", root, "rev-parse", "HEAD")
	revision, err := command.Output()
	if err != nil {
		return "", errors.New("Git revision could not be captured")
	}
	return strings.TrimSpace(string(revision)), nil
}

// e07ProductionBaselineMatches proves that no production source differs from
// the recorded baseline commit. Test-only and documentation changes are outside
// the compared paths, so committing the E07 change set keeps a future live pass
// runnable.
func E07ProductionBaselineMatches(root, baseline string) (bool, error) {
	args := append([]string{"-C", root, "diff", "--quiet", baseline, "--"}, e07ProductionPathspecs()...)
	command := exec.Command("git", args...)
	err := command.Run()
	if err == nil {
		return E07NoUntrackedProductionFiles(root)
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, errors.New("production baseline comparison failed")
}

func E07NoUntrackedProductionFiles(root string) (bool, error) {
	args := append([]string{"-C", root, "ls-files", "--others", "--"}, e07ProductionPathspecs()...)
	command := exec.Command("git", args...)
	untracked, err := command.Output()
	if err != nil {
		return false, errors.New("production baseline untracked check failed")
	}
	return len(bytes.TrimSpace(untracked)) == 0, nil
}

func e07ProductionPathspecs() []string {
	return []string{
		"apps/backend/internal",
		"apps/backend/cmd/ai-hub/main.go",
		"apps/backend/go.mod",
		"apps/backend/go.sum",
		":(glob)apps/backend/cmd/ai-hub/lifecycle/*.go",
		":(exclude,glob)apps/backend/cmd/ai-hub/lifecycle/*_test.go",
	}
}

func E07HashFile(hasher io.Writer, name, path string, info os.FileInfo) error {
	file, err := os.Open(path)
	if err != nil {
		return errors.New("integrity fingerprint could not read a file")
	}
	defer func() { _ = file.Close() }()
	_, _ = fmt.Fprintf(hasher, "file:%s:%o:%d:", name, info.Mode().Perm(), info.Size())
	if _, err := io.Copy(hasher, file); err != nil {
		return errors.New("integrity fingerprint could not hash a file")
	}
	_, _ = io.WriteString(hasher, "\n")
	return nil
}
