package integrity

import (
	e07support "aihub/cmd/ai-hub/e07/support"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestE07RepositoryRootResolvesFromNestedWorkingDirectory(t *testing.T) {
	e07support.E07GitAvailable(t)

	root := t.TempDir()
	e07support.E07RunGit(t, root, "init", "-q")
	nested := filepath.Join(root, "cmd", "ai-hub")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal("nested fixture directory could not be created")
	}

	resolved, err := e07support.E07RepositoryRoot(nested)
	if err != nil {
		t.Fatalf("repository root resolution failed: %v", err)
	}
	expected, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("fixture root could not be resolved: %v", err)
	}
	if resolved != expected {
		t.Fatalf("repository root was not the Git toplevel")
	}
}

func TestE07RepositoryRootFallsBackToModuleMarker(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal("module marker could not be written")
	}
	nested := filepath.Join(root, "cmd", "ai-hub")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal("nested fixture directory could not be created")
	}

	resolved, err := e07support.E07RepositoryRoot(nested)
	if err != nil {
		t.Fatalf("module-marker root resolution failed: %v", err)
	}
	if resolved != root {
		t.Fatal("repository root was not the nearest module marker")
	}
}

func TestE07RepositoryRootRejectsUnknownLocation(t *testing.T) {
	root := t.TempDir()
	if _, err := e07support.E07RepositoryRoot(root); err == nil {
		t.Fatal("an arbitrary directory was accepted as the repository root")
	}
}

func TestE07EnvFilesCoverRootAndNestedLocations(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		".env":              "root",
		".env.example":      "root-example",
		"nested/.env":       "nested",
		"nested/.env.local": "nested-local",
		"nested/env.txt":    "ignored",
	}
	for relative, content := range files {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal("environment-file fixture directory could not be created")
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal("environment-file fixture could not be written")
		}
	}

	discovered, err := e07support.E07EnvFiles(root)
	if err != nil {
		t.Fatalf("environment-file discovery failed: %v", err)
	}
	want := map[string]bool{".env": true, "nested/.env": true, "nested/.env.local": true}
	if len(discovered) != len(want) {
		t.Fatalf("discovered %d environment files, want %d", len(discovered), len(want))
	}
	for _, relative := range discovered {
		if !want[relative] {
			t.Fatalf("environment-file discovery returned an unexpected entry")
		}
	}
}

func TestE07RepositorySnapshotDetectsEveryRelevantChange(t *testing.T) {
	root := e07support.E07IntegrityFixture(t)

	baseline, err := e07support.E07SnapshotRepository(root)
	if err != nil {
		t.Fatalf("repository baseline could not be captured: %v", err)
	}
	repeated, err := e07support.E07SnapshotRepository(root)
	if err != nil {
		t.Fatalf("repeated repository baseline could not be captured: %v", err)
	}
	if comparison := e07support.E07CompareIntegrity(baseline, repeated); !comparison.Tree || !comparison.Env || !comparison.Tracked || !comparison.GitState {
		t.Fatalf("an unchanged repository produced an integrity difference: %s", comparison.Summary())
	}

	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("version-three\n"), 0o600); err != nil {
		t.Fatal("pre-dirty tracked file could not be changed")
	}
	dirtyTree, err := e07support.E07SnapshotRepository(root)
	if err != nil {
		t.Fatalf("repository snapshot after a tracked change failed: %v", err)
	}
	comparison := e07support.E07CompareIntegrity(baseline, dirtyTree)
	if comparison.Tree || comparison.Tracked {
		t.Fatalf("a content change to an already-dirty tracked file was not detected: %s", comparison.Summary())
	}
	if !comparison.GitState {
		t.Fatal("Git status equality masked the pre-dirty content change")
	}

	baseline = dirtyTree
	if err := os.WriteFile(filepath.Join(root, "untracked.txt"), []byte("untracked-two\n"), 0o600); err != nil {
		t.Fatal("untracked file could not be changed")
	}
	untrackedSnapshot, err := e07support.E07SnapshotRepository(root)
	if err != nil {
		t.Fatalf("repository snapshot after an untracked change failed: %v", err)
	}
	comparison = e07support.E07CompareIntegrity(baseline, untrackedSnapshot)
	if comparison.Tree {
		t.Fatalf("an untracked content change was not detected: %s", comparison.Summary())
	}
	if !comparison.Tracked {
		t.Fatal("an untracked file changed the tracked-file digest")
	}

	baseline = untrackedSnapshot
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("ROOT_SETTING=two\n"), 0o600); err != nil {
		t.Fatal("root environment file could not be changed")
	}
	envSnapshot, err := e07support.E07SnapshotRepository(root)
	if err != nil {
		t.Fatalf("repository snapshot after an environment change failed: %v", err)
	}
	comparison = e07support.E07CompareIntegrity(baseline, envSnapshot)
	if comparison.Env {
		t.Fatal("a root .env content change was not detected")
	}

	baseline = envSnapshot
	symlinkTarget, err := os.Readlink(filepath.Join(root, "state-link"))
	if err != nil {
		t.Fatalf("fixture symbolic link could not be read: %v", err)
	}
	if strings.HasPrefix(symlinkTarget, root) {
		t.Fatal("fixture symbolic link target must be outside the repository")
	}
	if err := os.WriteFile(symlinkTarget, []byte("target-two\n"), 0o600); err != nil {
		t.Fatal("symbolic-link target could not be changed")
	}
	linkSnapshot, err := e07support.E07SnapshotRepository(root)
	if err != nil {
		t.Fatalf("repository snapshot after a symbolic-link target change failed: %v", err)
	}
	comparison = e07support.E07CompareIntegrity(baseline, linkSnapshot)
	if comparison.Tree {
		t.Fatal("a nested symbolic-link target content change was not detected")
	}
}
