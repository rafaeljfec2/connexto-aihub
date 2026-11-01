package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestE07IntegrityManifestIsPrivateAndRoundTrips(t *testing.T) {
	root := e07IntegrityFixture(t)
	snapshot, err := e07SnapshotRepository(root)
	if err != nil {
		t.Fatalf("repository snapshot failed: %v", err)
	}

	privateDir := t.TempDir()
	path, err := e07SaveIntegrityManifest(privateDir, snapshot)
	if err != nil {
		t.Fatalf("private integrity manifest could not be written: %v", err)
	}
	if relative, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(relative, "..") {
		t.Fatal("private integrity manifest was written inside the repository")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("private integrity manifest could not be inspected: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("private integrity manifest mode = %o, want 600", mode)
	}

	loaded, err := e07LoadIntegrityManifest(path)
	if err != nil {
		t.Fatalf("private integrity manifest could not be read: %v", err)
	}
	if comparison := e07CompareIntegrity(snapshot, loaded); !comparison.Tree || !comparison.Env || !comparison.Tracked || !comparison.GitState {
		t.Fatalf("manifest round trip changed the snapshot: %s", comparison.Summary())
	}
}

func TestE07IntegritySummaryIsAggregatedAndRedacted(t *testing.T) {
	root := e07IntegrityFixture(t)
	snapshot, err := e07SnapshotRepository(root)
	if err != nil {
		t.Fatalf("repository snapshot failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("leaked-content\n"), 0o600); err != nil {
		t.Fatal("fixture file could not be changed")
	}
	changed, err := e07SnapshotRepository(root)
	if err != nil {
		t.Fatalf("changed repository snapshot failed: %v", err)
	}

	summary := e07CompareIntegrity(snapshot, changed).Summary()
	for _, forbidden := range []string{root, "tracked.txt", "leaked-content", snapshot.Tree, snapshot.Tracked} {
		if forbidden != "" && strings.Contains(summary, forbidden) {
			t.Fatalf("integrity summary exposes private evidence")
		}
	}
	if !strings.Contains(summary, "repository_tree=FAIL") || !strings.Contains(summary, "git_scope=PASS") {
		t.Fatalf("integrity summary is not aggregated by category: %s", summary)
	}
}

func TestE07PrivateManifestDirectoryRejectsRepositoryLocations(t *testing.T) {
	root := e07IntegrityFixture(t)

	if _, err := e07PrivateManifestDirectory(root, filepath.Join(root, "nested")); err == nil {
		t.Fatal("a private manifest directory inside the repository was accepted")
	}

	external := t.TempDir()
	directory, err := e07PrivateManifestDirectory(root, external)
	if err != nil {
		t.Fatalf("an external private manifest directory was rejected: %v", err)
	}
	relative, err := filepath.Rel(root, directory)
	if err != nil {
		t.Fatalf("private manifest directory could not be related to the repository: %v", err)
	}
	if relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatal("private manifest directory resolved inside the repository")
	}
}

func TestE07ProductionBaselineComparisonIgnoresTestOnlyChanges(t *testing.T) {
	e07GitAvailable(t)

	root := t.TempDir()
	e07RunGit(t, root, "init", "-q")
	files := map[string]string{
		"go.mod":                       "module fixture\n\ngo 1.23\n",
		"internal/production.go":       "package internal\n",
		"cmd/ai-hub/main.go":           "package main\n",
		"cmd/ai-hub/e07_scope_test.go": "package main\n",
	}
	for relative, content := range files {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal("baseline fixture directory could not be created")
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal("baseline fixture file could not be written")
		}
	}
	e07RunGit(t, root, "add", ".")
	e07RunGit(t, root, "commit", "-q", "-m", "baseline")
	baseline, err := e07GitRevision(root)
	if err != nil {
		t.Fatalf("baseline revision could not be captured: %v", err)
	}

	matches, err := e07ProductionBaselineMatches(root, baseline)
	if err != nil || !matches {
		t.Fatalf("unchanged production source did not match the baseline: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "cmd", "ai-hub", "e07_scope_test.go"), []byte("package main\n\n// test-only change\n"), 0o600); err != nil {
		t.Fatal("test-only fixture file could not be changed")
	}
	matches, err = e07ProductionBaselineMatches(root, baseline)
	if err != nil || !matches {
		t.Fatalf("a test-only change affected the production baseline: %v", err)
	}

	if err := os.WriteFile(filepath.Join(root, "internal", "production.go"), []byte("package internal\n\n// production change\n"), 0o600); err != nil {
		t.Fatal("production fixture file could not be changed")
	}
	matches, err = e07ProductionBaselineMatches(root, baseline)
	if err != nil {
		t.Fatalf("changed production source could not be compared: %v", err)
	}
	if matches {
		t.Fatal("a production source change matched the baseline")
	}

	if err := os.WriteFile(filepath.Join(root, "internal", "untracked_production.go"), []byte("package internal\n"), 0o600); err != nil {
		t.Fatal("untracked production fixture file could not be written")
	}
	e07RunGit(t, root, "checkout", "--", "internal/production.go")
	matches, err = e07ProductionBaselineMatches(root, baseline)
	if err != nil {
		t.Fatalf("untracked production source could not be compared: %v", err)
	}
	if matches {
		t.Fatal("an untracked production source file matched the baseline")
	}

	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("internal/ignored_production.go\n"), 0o600); err != nil {
		t.Fatal("baseline fixture ignore file could not be written")
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "ignored_production.go"), []byte("package internal\n"), 0o600); err != nil {
		t.Fatal("ignored production fixture file could not be written")
	}
	if err := os.Remove(filepath.Join(root, "internal", "untracked_production.go")); err != nil {
		t.Fatal("untracked production fixture file could not be removed")
	}
	matches, err = e07ProductionBaselineMatches(root, baseline)
	if err != nil {
		t.Fatalf("ignored production source could not be compared: %v", err)
	}
	if matches {
		t.Fatal("an ignored production source file matched the baseline")
	}
}

func TestE07FingerprintTreeTerminatesOnSymbolicLinkCycle(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "state"), []byte("content"), 0o600); err != nil {
		t.Fatal("cycle fixture file could not be written")
	}
	if err := os.Symlink(root, filepath.Join(root, "loop")); err != nil {
		t.Skip("symbolic links are unavailable")
	}

	first, err := e07FingerprintTree(root)
	if err != nil {
		t.Fatalf("cycle fingerprint failed: %v", err)
	}
	second, err := e07FingerprintTree(root)
	if err != nil {
		t.Fatalf("repeated cycle fingerprint failed: %v", err)
	}
	if first != second {
		t.Fatal("symbolic-link cycle produced a non-deterministic fingerprint")
	}
}
