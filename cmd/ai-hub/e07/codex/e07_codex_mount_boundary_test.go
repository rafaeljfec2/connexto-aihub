package codex

import (
	e07support "aihub/cmd/ai-hub/e07/support"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestE07CodexMountBoundaryHidesGlobalState(t *testing.T) {
	mode := os.Getenv(e07support.E07MountBoundaryModeEnv)
	if mode != "" {
		e07support.E07VerifyCodexMountBoundary(t, mode)
		return
	}
	if runtime.GOOS != "linux" {
		t.Skip("private Codex mount boundary requires Linux")
	}
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		t.Skip("bubblewrap is unavailable")
	}
	globalCodex, err := e07support.E07AccountCodexHome()
	if err != nil {
		t.Fatal("account Codex state root could not be resolved")
	}
	privateRoot := t.TempDir()
	hiddenRoot := filepath.Join(privateRoot, "hidden-codex")
	if err := os.Mkdir(hiddenRoot, 0o700); err != nil {
		t.Fatal("empty Codex mount source could not be created")
	}
	privateCodex := filepath.Join(privateRoot, "codex-home")
	for _, directory := range []string{
		privateCodex, filepath.Join(privateRoot, "home"), filepath.Join(privateRoot, "tmp"),
		filepath.Join(privateRoot, "xdg", "config"), filepath.Join(privateRoot, "xdg", "data"),
		filepath.Join(privateRoot, "xdg", "cache"), filepath.Join(privateRoot, "xdg", "state"),
		filepath.Join(privateRoot, "xdg", "runtime"),
	} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal("private Codex runtime directory could not be created")
		}
	}
	if err := os.Chmod(filepath.Join(privateRoot, "xdg", "runtime"), 0o700); err != nil {
		t.Fatal("private runtime directory permissions could not be set")
	}
	beforePrivate, err := e07support.E07SnapshotPrivateCodexState(privateCodex)
	if err != nil || !e07support.E07PrivateCodexStateSnapshotValid(beforePrivate, len(e07support.E07CodexRuntimeDirectories(privateCodex))) {
		t.Fatal("private Codex state could not be captured before execution")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal("test executable could not be resolved")
	}
	repository, err := e07support.E07RepositoryRoot("")
	if err != nil {
		t.Fatal("repository root could not be resolved")
	}
	e07support.E07CopyTestExecutable(t, executable, privateRoot)
	args := e07support.E07CodexBoundaryArgs(bwrap, hiddenRoot, globalCodex, privateRoot, repository,
		filepath.Join(e07support.E07PrivateMountRoot, "e07-test-binary"))
	command := exec.Command(args[0], args[1:]...)
	command.Env = []string{"PATH=" + os.Getenv("PATH")}
	if _, err := command.CombinedOutput(); err != nil {
		t.Fatal("private Codex mount boundary or descendant verification failed")
	}
	afterPrivate, err := e07support.E07SnapshotPrivateCodexState(privateCodex)
	if err != nil || !e07support.E07PrivateCodexStateSnapshotValid(afterPrivate, len(e07support.E07CodexRuntimeDirectories(privateCodex))) {
		t.Fatal("private Codex state could not be captured after execution")
	}
	for _, marker := range []string{"parent-state", "child-state"} {
		if _, err := os.Stat(filepath.Join(privateCodex, marker)); err != nil {
			t.Fatalf("private Codex state was not verifiable after execution: %s", marker)
		}
	}
}
