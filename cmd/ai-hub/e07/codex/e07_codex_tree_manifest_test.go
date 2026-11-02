package codex

import (
	e07support "aihub/cmd/ai-hub/e07/support"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestE07CodexTreeSnapshotCoversConfiguredStateRoots(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o700); err != nil {
		t.Fatal("default Codex state fixture could not be created")
	}
	if err := os.WriteFile(filepath.Join(home, ".codex", "marker"), []byte("synthetic metadata"), 0o600); err != nil {
		t.Fatal("default Codex state fixture marker could not be created")
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	stateRoots := map[string]string{
		"CODEX_HOME":        filepath.Join(root, "codex-home"),
		"CODEX_SQLITE_HOME": filepath.Join(root, "sqlite-home"),
	}
	for variable, path := range stateRoots {
		t.Setenv(variable, path)
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal("Codex state fixture could not be created")
		}
		if err := os.WriteFile(filepath.Join(path, "marker"), []byte("synthetic metadata"), 0o600); err != nil {
			t.Fatal("Codex state fixture marker could not be created")
		}
	}

	entries, err := e07support.E07SnapshotCodexTreeAt(e07support.E07CodexTreeRootsFor(home, home))
	if err != nil {
		t.Fatal("Codex state roots could not be captured")
	}
	seen := make(map[string]bool, len(stateRoots)+1)
	for _, entry := range entries {
		if entry.Path == "marker" {
			seen[entry.Root] = entry.Metadata != ""
		}
	}
	for _, name := range []string{"default_codex_home", "codex_home", "codex_sqlite_home"} {
		if !seen[name] {
			t.Fatalf("Codex state root %q is missing from the private metadata snapshot", name)
		}
	}
	for _, entry := range entries {
		if entry.Root == "process_home_codex_home" {
			t.Fatal("identical account and process homes were not deduplicated")
		}
	}
}

func TestE07CodexTreeRootsResolveAccountHomeWhenProcessHomeIsIsolated(t *testing.T) {
	isolated := t.TempDir()
	t.Setenv("HOME", isolated)
	t.Setenv("USERPROFILE", isolated)
	t.Setenv("CODEX_HOME", "")
	t.Setenv("CODEX_SQLITE_HOME", "")

	accountHome, err := e07support.E07AccountHomeDirectory()
	if err != nil {
		t.Skip("account home is unavailable in this environment")
	}
	roots, err := e07support.E07CodexTreeRoots()
	if err != nil {
		t.Fatal("Codex state roots could not be resolved")
	}
	for _, root := range roots {
		if root.Name != "default_codex_home" {
			continue
		}
		if root.Path != filepath.Join(accountHome, ".codex") {
			t.Fatal("resolved Codex state roots did not include the account home")
		}

		return
	}
	t.Fatal("resolved Codex state roots omitted the account Codex home")
}

func TestE07CodexTreeRootsTrackOnlyCodexManagedLocations(t *testing.T) {
	root := t.TempDir()
	accountHome := filepath.Join(root, "account-home")
	processHome := filepath.Join(root, "process-home")
	codexHome := filepath.Join(root, "codex-home")
	sqliteHome := filepath.Join(root, "sqlite-home")
	for _, variable := range []string{
		"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME", "XDG_RUNTIME_DIR",
	} {
		t.Setenv(variable, filepath.Join(root, variable))
	}
	t.Setenv("CODEX_HOME", codexHome)
	t.Setenv("CODEX_SQLITE_HOME", sqliteHome)

	roots := e07support.E07CodexTreeRootsFor(accountHome, processHome)
	expected := map[string]string{
		"default_codex_home":      filepath.Join(accountHome, ".codex"),
		"process_home_codex_home": filepath.Join(processHome, ".codex"),
		"codex_home":              codexHome,
		"codex_sqlite_home":       sqliteHome,
	}
	if len(roots) != len(expected) {
		t.Fatal("Codex tree snapshot included non-Codex XDG roots")
	}
	for _, stateRoot := range roots {
		if expected[stateRoot.Name] != stateRoot.Path {
			t.Fatalf("Codex tree snapshot included an unexpected root category: %s", stateRoot.Name)
		}
	}
}

func TestE07CodexTreeSnapshotKeepsAccountRootWhenProcessHomeIsIsolated(t *testing.T) {
	root := t.TempDir()
	accountHome := filepath.Join(root, "account-home")
	isolatedHome := filepath.Join(root, "isolated-home")
	if err := os.MkdirAll(filepath.Join(accountHome, ".codex"), 0o700); err != nil {
		t.Fatal("account Codex state fixture could not be created")
	}
	if err := os.WriteFile(filepath.Join(accountHome, ".codex", "marker"), []byte("synthetic metadata"), 0o600); err != nil {
		t.Fatal("account Codex state fixture marker could not be created")
	}
	t.Setenv("HOME", isolatedHome)
	t.Setenv("CODEX_HOME", "")
	t.Setenv("CODEX_SQLITE_HOME", "")

	entries, err := e07support.E07SnapshotCodexTreeAt(e07support.E07CodexTreeRootsFor(accountHome, isolatedHome))
	if err != nil {
		t.Fatal("Codex state roots could not be captured with an isolated process home")
	}
	for _, entry := range entries {
		if entry.Root == "default_codex_home" && entry.Path == "marker" {
			return
		}
	}
	t.Fatal("isolated process home hid the account Codex state root")
}

func TestE07CodexTreeMetadataExcludesDirectoryAccessTimes(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the Codex metadata representation is platform-specific")
	}
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o700); err != nil {
		t.Fatal("temporary Codex home could not be created")
	}
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	t.Setenv("CODEX_SQLITE_HOME", "")

	entries, err := e07support.E07SnapshotCodexTreeAt(e07support.E07CodexTreeRootsFor(home, home))
	if err != nil {
		t.Fatal("temporary Codex metadata could not be captured")
	}
	for _, entry := range entries {
		if strings.Contains(entry.Metadata, "Atim") || strings.Contains(entry.Metadata, "Atime") {
			t.Fatal("Codex metadata includes access times that can change during traversal")
		}
	}
}

func TestE07CodexTreeSnapshotRejectsSymlinkStateRoot(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "state-target")
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal("synthetic Codex state target could not be created")
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal("synthetic home could not be created")
	}
	if err := os.Symlink(target, filepath.Join(home, ".codex")); err != nil {
		t.Skip("symbolic links are unavailable in this test environment")
	}
	e07support.E07SetCodexSnapshotHome(t, home)

	if _, err := e07support.E07SnapshotCodexTreeAt(e07support.E07CodexTreeRootsFor(home, home)); err == nil {
		t.Fatal("Codex tree snapshot accepted a symbolic-link state root")
	}
}

func TestE07CodexTreeSnapshotRecordsNestedSymlink(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	codexHome := filepath.Join(home, ".codex")
	target := filepath.Join(root, "state-target")
	otherTarget := filepath.Join(root, "other-target")
	if err := os.MkdirAll(codexHome, 0o700); err != nil {
		t.Fatal("synthetic Codex home could not be created")
	}
	for _, directory := range []string{target, otherTarget} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal("synthetic state target could not be created")
		}
	}
	link := filepath.Join(codexHome, "linked-state")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symbolic links are unavailable in this test environment")
	}
	e07support.E07SetCodexSnapshotHome(t, home)
	roots := e07support.E07CodexTreeRootsFor(home, home)

	entries, err := e07support.E07SnapshotCodexTreeAt(roots)
	if err != nil {
		t.Fatal("Codex tree snapshot rejected a nested symbolic link")
	}
	linked := false
	for _, entry := range entries {
		if entry.Path == "linked-state" {
			linked = entry.Mode&os.ModeSymlink != 0 && strings.Contains(entry.Metadata, "|target="+target)
		}
	}
	if !linked {
		t.Fatal("Codex tree snapshot did not record the nested symbolic-link target")
	}
	if count := e07support.E07CodexTreeLinkCount(entries); count != 1 {
		t.Fatal("Codex tree snapshot did not report the nested symbolic-link coverage")
	}

	if err := os.Remove(link); err != nil {
		t.Fatal("synthetic symbolic link could not be replaced")
	}
	if err := os.Symlink(otherTarget, link); err != nil {
		t.Fatal("synthetic symbolic link could not be retargeted")
	}
	retargeted, err := e07support.E07SnapshotCodexTreeAt(roots)
	if err != nil {
		t.Fatal("Codex tree snapshot could not be re-captured after a link change")
	}
	if e07support.E07CodexTreeUnchanged(entries, retargeted) {
		t.Fatal("Codex tree comparison ignored a nested symbolic-link change")
	}
}
