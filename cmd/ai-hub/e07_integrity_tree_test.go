package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// e07FingerprintTree hashes every entry below a path. Symbolic links record
// their target and, when resolvable, the digest of the target content so a
// change behind a nested link is detected. Directory links are followed with
// cycle protection.
func e07FingerprintTree(path string) (string, error) {
	hasher := sha256.New()
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		_, _ = io.WriteString(hasher, "absent")

		return hex.EncodeToString(hasher.Sum(nil)), nil
	}
	if err != nil {
		return "", errors.New("integrity fingerprint could not inspect a path")
	}
	entries := 0
	if err := e07DigestPath(hasher, path, "", info, make(map[string]struct{}), &entries); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func e07DigestPath(hasher io.Writer, path, name string, info os.FileInfo, visiting map[string]struct{}, entries *int) error {
	*entries++
	if *entries > e07IntegrityMaxEntries {
		return errors.New("integrity fingerprint exceeded the validation bound")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return e07DigestSymlink(hasher, path, name, info, visiting, entries)
	}
	if info.IsDir() {
		return e07DigestDirectory(hasher, path, name, info, visiting, entries)
	}
	if !info.Mode().IsRegular() {
		_, _ = fmt.Fprintf(hasher, "special:%s:%o:%d\n", name, info.Mode(), info.Size())

		return nil
	}

	return e07HashFile(hasher, name, path, info)
}

func e07DigestDirectory(hasher io.Writer, path, name string, info os.FileInfo, visiting map[string]struct{}, entries *int) error {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return errors.New("integrity fingerprint could not resolve a directory")
	}
	if _, cycle := visiting[resolved]; cycle {
		_, _ = fmt.Fprintf(hasher, "cycle:%s\n", name)

		return nil
	}
	visiting[resolved] = struct{}{}
	defer delete(visiting, resolved)

	_, _ = fmt.Fprintf(hasher, "dir:%s:%o\n", name, info.Mode().Perm())
	children, err := os.ReadDir(path)
	if err != nil {
		return errors.New("integrity fingerprint could not traverse a path")
	}
	for _, child := range children {
		if child.IsDir() && child.Name() == ".git" {
			continue
		}
		childPath := filepath.Join(path, child.Name())
		childName := child.Name()
		if name != "" {
			childName = name + "/" + child.Name()
		}
		childInfo, err := os.Lstat(childPath)
		if err != nil {
			return errors.New("integrity fingerprint could not inspect an entry")
		}
		if err := e07DigestPath(hasher, childPath, childName, childInfo, visiting, entries); err != nil {
			return err
		}
	}

	return nil
}

func e07DigestSymlink(hasher io.Writer, path, name string, info os.FileInfo, visiting map[string]struct{}, entries *int) error {
	target, err := os.Readlink(path)
	if err != nil {
		return errors.New("integrity fingerprint could not inspect a symbolic link")
	}
	_, _ = fmt.Fprintf(hasher, "link:%s:%o:%s\n", name, info.Mode().Perm(), target)

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		_, _ = fmt.Fprintf(hasher, "link-unresolved:%s\n", name)

		return nil
	}
	resolvedInfo, err := os.Stat(resolved)
	if err != nil {
		_, _ = fmt.Fprintf(hasher, "link-unreadable:%s\n", name)

		return nil
	}

	return e07DigestPath(hasher, resolved, name+"->", resolvedInfo, visiting, entries)
}

func e07HashFile(hasher io.Writer, name, path string, info os.FileInfo) error {
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

// e07FingerprintTrackedFiles hashes the content of every Git-tracked file,
// including files that were already modified before the validation started.
func e07FingerprintTrackedFiles(root string) (string, error) {
	files, err := e07GitTrackedFiles(root)
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
		if err := e07DigestPath(hasher, path, relative, info, make(map[string]struct{}), new(int)); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func e07GitTrackedFiles(root string) ([]string, error) {
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

func e07GitState(root string) (string, error) {
	command := exec.Command("git", "-C", root, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	state, err := command.Output()
	if err != nil {
		return "", errors.New("Git worktree state could not be captured")
	}
	digest := sha256.Sum256(state)

	return hex.EncodeToString(digest[:]), nil
}

func e07GitRevision(root string) (string, error) {
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
func e07ProductionBaselineMatches(root, baseline string) (bool, error) {
	command := exec.Command("git", "-C", root, "diff", "--quiet", baseline, "--",
		"internal", "cmd/ai-hub/main.go", "cmd/ai-hub/lifecycle.go", "go.mod", "go.sum")
	err := command.Run()
	if err == nil {
		return e07NoUntrackedProductionFiles(root)
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}

	return false, errors.New("production baseline comparison failed")
}

func e07NoUntrackedProductionFiles(root string) (bool, error) {
	command := exec.Command("git", "-C", root, "ls-files", "--others", "--",
		"internal", "cmd/ai-hub/main.go", "cmd/ai-hub/lifecycle.go", "go.mod", "go.sum")
	untracked, err := command.Output()
	if err != nil {
		return false, errors.New("production baseline untracked check failed")
	}

	return len(bytes.TrimSpace(untracked)) == 0, nil
}

// e07ErrorEnvelopePresent classifies a generic upstream error envelope without
// retaining its content.
func e07ErrorEnvelopePresent(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	var envelope struct {
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return false
	}
	value := bytes.TrimSpace(envelope.Error)

	return len(value) > 0 && !bytes.Equal(value, []byte("null"))
}
