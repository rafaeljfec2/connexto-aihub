package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	e07IntegrityMaxEntries = 250_000
	e07BaselineRevision    = "0d33ca26c15acf54fac418cb33380b5871b4d899"
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
