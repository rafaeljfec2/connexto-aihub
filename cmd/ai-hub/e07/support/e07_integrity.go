package support

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
	E07IntegrityMaxEntries = 250_000
	E07BaselineRevision    = "0d33ca26c15acf54fac418cb33380b5871b4d899"
)

// e07IntegritySnapshot holds aggregate digests only. It never stores paths,
// file contents, or raw status output.
type E07IntegritySnapshot struct {
	Tree     string `json:"tree"`
	Env      string `json:"env"`
	Tracked  string `json:"tracked"`
	GitState string `json:"git_state"`
}

// e07IntegrityComparison reports each integrity category independently.
type E07IntegrityComparison struct {
	Tree     bool
	Env      bool
	Tracked  bool
	GitState bool
}

func (comparison E07IntegrityComparison) Summary() string {
	return "repository_tree=" + E07IntegrityResult(comparison.Tree) +
		" env_files=" + E07IntegrityResult(comparison.Env) +
		" tracked_files=" + E07IntegrityResult(comparison.Tracked) +
		" git_scope=" + E07IntegrityResult(comparison.GitState)
}

func E07IntegrityResult(passed bool) string {
	if passed {
		return "PASS"
	}
	return "FAIL"
}

// e07RepositoryRoot resolves the repository root from Git metadata, falling
// back to the nearest module marker. It never assumes the Go test runner's
// package working directory is the repository root.
func E07RepositoryRoot(start string) (string, error) {
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
	if toplevel, err := E07GitToplevel(absolute); err == nil {
		return E07ResolveDirectory(toplevel)
	}
	for current := absolute; ; current = filepath.Dir(current) {
		if info, err := os.Stat(filepath.Join(current, "go.mod")); err == nil && !info.IsDir() {
			return E07ResolveDirectory(current)
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}

	return "", errors.New("repository root could not be resolved")
}

func E07GitToplevel(dir string) (string, error) {
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

func E07ResolveDirectory(path string) (string, error) {
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

// e07EnvFiles lists secret-bearing dotenv files without reading their contents.
func E07EnvFiles(root string) ([]string, error) {
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
		info, err := os.Lstat(path)
		if err != nil {
			return errors.New("environment-file metadata could not be inspected")
		}
		if !E07IsSecretEnvironmentEntry(path, entry.Name(), info) {
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

func E07FingerprintEnvFiles(root string, files []string) (string, error) {
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
		if err := E07DigestPath(hasher, path, relative, info, make(map[string]struct{}), new(int)); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// e07SnapshotRepository captures source content, dotenv metadata, and Git state
// without reading secret-bearing dotenv files.
func E07SnapshotRepository(root string) (E07IntegritySnapshot, error) {
	tree, err := E07FingerprintTree(root)
	if err != nil {
		return E07IntegritySnapshot{}, err
	}
	envFiles, err := E07EnvFiles(root)
	if err != nil {
		return E07IntegritySnapshot{}, err
	}
	env, err := E07FingerprintEnvFiles(root, envFiles)
	if err != nil {
		return E07IntegritySnapshot{}, err
	}
	tracked, err := E07FingerprintTrackedFiles(root)
	if err != nil {
		return E07IntegritySnapshot{}, err
	}
	gitState, err := E07GitState(root)
	if err != nil {
		return E07IntegritySnapshot{}, err
	}

	return E07IntegritySnapshot{Tree: tree, Env: env, Tracked: tracked, GitState: gitState}, nil
}

func E07CompareIntegrity(before, after E07IntegritySnapshot) E07IntegrityComparison {
	return E07IntegrityComparison{
		Tree:     before.Tree == after.Tree,
		Env:      before.Env == after.Env,
		Tracked:  before.Tracked == after.Tracked,
		GitState: before.GitState == after.GitState,
	}
}
