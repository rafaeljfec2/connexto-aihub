package support

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func E07GitAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("Git is unavailable")
	}
}

func E07RunGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", dir}, args...)...)
	command.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=e07",
		"GIT_AUTHOR_EMAIL=e07@example.invalid",
		"GIT_COMMITTER_NAME=e07",
		"GIT_COMMITTER_EMAIL=e07@example.invalid",
	)
	if _, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v", args, err)
	}
}

// e07IntegrityFixture builds a temporary repository with a root .env, a nested
// .env file, a tracked file that is already modified, an untracked file, and a
// symbolic link to a regular file.
func E07IntegrityFixture(t *testing.T) string {
	t.Helper()
	E07GitAvailable(t)

	root := t.TempDir()
	E07RunGit(t, root, "init", "-q")

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatal("fixture module marker could not be written")
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("ROOT_SETTING=one\n"), 0o600); err != nil {
		t.Fatal("fixture root environment file could not be written")
	}
	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal("fixture nested directory could not be created")
	}
	if err := os.WriteFile(filepath.Join(nested, ".env.local"), []byte("NESTED_SETTING=one\n"), 0o600); err != nil {
		t.Fatal("fixture nested environment file could not be written")
	}
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("version-one\n"), 0o600); err != nil {
		t.Fatal("fixture tracked file could not be written")
	}
	E07RunGit(t, root, "add", "go.mod", ".env", "nested/.env.local", "tracked.txt")

	// The tracked file is already dirty before the baseline is captured.
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("version-two\n"), 0o600); err != nil {
		t.Fatal("fixture tracked file could not be modified")
	}
	if err := os.WriteFile(filepath.Join(root, "untracked.txt"), []byte("untracked-one\n"), 0o600); err != nil {
		t.Fatal("fixture untracked file could not be written")
	}
	target := filepath.Join(t.TempDir(), "link-target.txt")
	if err := os.WriteFile(target, []byte("target-one\n"), 0o600); err != nil {
		t.Fatal("fixture symbolic-link target could not be written")
	}
	if err := os.Symlink(target, filepath.Join(root, "state-link")); err != nil {
		t.Skip("symbolic links are unavailable")
	}

	return root
}
