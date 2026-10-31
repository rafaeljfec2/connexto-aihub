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

const (
	e07IntegrityMaxEntries      = 250_000
	e07IntegrityManifestName    = "e07-integrity-manifest.json"
	e07IntegrityManifestPerm    = 0o600
	e07IntegrityManifestDirPerm = 0o700
	e07BaselineRevision         = "0d33ca26c15acf54fac418cb33380b5871b4d899"
)

// e07IntegritySnapshot holds aggregate digests only. It never stores paths,
// file contents, or raw status output.
type e07IntegritySnapshot struct {
	Tree     string `json:"tree"`
	Env      string `json:"env"`
	Tracked  string `json:"tracked"`
	GitState string `json:"git_state"`
}

// e07IntegrityComparison reports each integrity category independently.
type e07IntegrityComparison struct {
	Tree     bool
	Env      bool
	Tracked  bool
	GitState bool
}

func (comparison e07IntegrityComparison) Summary() string {
	return "repository_tree=" + e07IntegrityResult(comparison.Tree) +
		" env_files=" + e07IntegrityResult(comparison.Env) +
		" tracked_files=" + e07IntegrityResult(comparison.Tracked) +
		" git_scope=" + e07IntegrityResult(comparison.GitState)
}

func e07IntegrityResult(passed bool) string {
	if passed {
		return "PASS"
	}
	return "FAIL"
}

// e07RepositoryRoot resolves the repository root from Git metadata, falling
// back to the nearest module marker. It never assumes the Go test runner's
// package working directory is the repository root.
func e07RepositoryRoot(start string) (string, error) {
	base := start
	if base == "" {
		workingDirectory, err := os.Getwd()
		if err != nil {
			return "", errors.New("repository root could not be resolved")
		}
		base = workingDirectory
	}
	absolute, err := filepath.Abs(base)
	if err != nil {
		return "", errors.New("repository root could not be resolved")
	}
	if info, err := os.Stat(absolute); err != nil || !info.IsDir() {
		return "", errors.New("repository root start path is unavailable")
	}
	if toplevel, err := e07GitToplevel(absolute); err == nil {
		return e07ResolveDirectory(toplevel)
	}
	for current := absolute; ; current = filepath.Dir(current) {
		if info, err := os.Stat(filepath.Join(current, "go.mod")); err == nil && !info.IsDir() {
			return e07ResolveDirectory(current)
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}

	return "", errors.New("repository root could not be resolved")
}

func e07GitToplevel(dir string) (string, error) {
	command := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	output, err := command.Output()
	if err != nil {
		return "", errors.New("Git toplevel could not be resolved")
	}
	root := strings.TrimSpace(string(output))
	if root == "" {
		return "", errors.New("Git toplevel was empty")
	}

	return root, nil
}

func e07ResolveDirectory(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", errors.New("integrity path could not be resolved")
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", errors.New("integrity path could not be resolved")
	}
	info, err := os.Stat(resolved)
	if err != nil || !info.IsDir() {
		return "", errors.New("integrity path is not a directory")
	}

	return resolved, nil
}

// e07EnvFiles lists every existing .env file below the repository root,
// including nested directories, as repository-relative slash paths.
func e07EnvFiles(root string) ([]string, error) {
	files := make([]string, 0, 4)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return errors.New("environment-file discovery could not traverse the project")
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}

			return nil
		}
		name := entry.Name()
		if name != ".env" && (!strings.HasPrefix(name, ".env.") || name == ".env.example") {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return errors.New("environment-file discovery could not name an entry")
		}
		files = append(files, filepath.ToSlash(relative))

		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)

	return files, nil
}

func e07FingerprintEnvFiles(root string, files []string) (string, error) {
	hasher := sha256.New()
	_, _ = fmt.Fprintf(hasher, "env-files:%d\n", len(files))
	for _, relative := range files {
		path := filepath.Join(root, filepath.FromSlash(relative))
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			_, _ = fmt.Fprintf(hasher, "missing:%s\n", relative)

			continue
		}
		if err != nil {
			return "", errors.New("environment-file fingerprint could not inspect an entry")
		}
		if err := e07DigestPath(hasher, path, relative, info, make(map[string]struct{}), new(int)); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// e07SnapshotRepository captures every category used by the integrity gate:
// the complete tree (tracked, already-dirty, and untracked files), every
// environment file, tracked-file content, and the Git status digest.
func e07SnapshotRepository(root string) (e07IntegritySnapshot, error) {
	tree, err := e07FingerprintTree(root)
	if err != nil {
		return e07IntegritySnapshot{}, err
	}
	envFiles, err := e07EnvFiles(root)
	if err != nil {
		return e07IntegritySnapshot{}, err
	}
	env, err := e07FingerprintEnvFiles(root, envFiles)
	if err != nil {
		return e07IntegritySnapshot{}, err
	}
	tracked, err := e07FingerprintTrackedFiles(root)
	if err != nil {
		return e07IntegritySnapshot{}, err
	}
	gitState, err := e07GitState(root)
	if err != nil {
		return e07IntegritySnapshot{}, err
	}

	return e07IntegritySnapshot{Tree: tree, Env: env, Tracked: tracked, GitState: gitState}, nil
}

func e07CompareIntegrity(before, after e07IntegritySnapshot) e07IntegrityComparison {
	return e07IntegrityComparison{
		Tree:     before.Tree == after.Tree,
		Env:      before.Env == after.Env,
		Tracked:  before.Tracked == after.Tracked,
		GitState: before.GitState == after.GitState,
	}
}

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
