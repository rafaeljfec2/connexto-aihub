package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

const (
	e07MountBoundaryModeEnv      = "AIHUB_E07_MOUNT_BOUNDARY_MODE"
	e07PrivateRootEnv            = "AIHUB_E07_PRIVATE_ROOT"
	e07ExpectedMountNamespaceEnv = "AIHUB_E07_EXPECTED_MOUNT_NAMESPACE"
	e07ExpectedPIDNamespaceEnv   = "AIHUB_E07_EXPECTED_PID_NAMESPACE"
	e07PrivateMountRoot          = "/run/e07/private"
)

func TestE07CodexMountBoundaryHidesGlobalState(t *testing.T) {
	mode := os.Getenv(e07MountBoundaryModeEnv)
	if mode != "" {
		e07VerifyCodexMountBoundary(t, mode)
		return
	}
	if runtime.GOOS != "linux" {
		t.Skip("private Codex mount boundary requires Linux")
	}
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		t.Skip("bubblewrap is unavailable")
	}
	globalCodex, err := e07AccountCodexHome()
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
	beforePrivate, err := e07SnapshotPrivateCodexState(privateCodex)
	if err != nil || !e07PrivateCodexStateSnapshotValid(beforePrivate, len(e07CodexRuntimeDirectories(privateCodex))) {
		t.Fatal("private Codex state could not be captured before execution")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal("test executable could not be resolved")
	}
	repository, err := e07RepositoryRoot("")
	if err != nil {
		t.Fatal("repository root could not be resolved")
	}
	e07CopyTestExecutable(t, executable, privateRoot)
	args := e07CodexBoundaryArgs(bwrap, hiddenRoot, globalCodex, privateRoot, repository,
		filepath.Join(e07PrivateMountRoot, "e07-test-binary"))
	command := exec.Command(args[0], args[1:]...)
	command.Env = []string{"PATH=" + os.Getenv("PATH")}
	if _, err := command.CombinedOutput(); err != nil {
		t.Fatal("private Codex mount boundary or descendant verification failed")
	}
	afterPrivate, err := e07SnapshotPrivateCodexState(privateCodex)
	if err != nil || !e07PrivateCodexStateSnapshotValid(afterPrivate, len(e07CodexRuntimeDirectories(privateCodex))) {
		t.Fatal("private Codex state could not be captured after execution")
	}
	for _, marker := range []string{"parent-state", "child-state"} {
		if _, err := os.Stat(filepath.Join(privateCodex, marker)); err != nil {
			t.Fatalf("private Codex state was not verifiable after execution: %s", marker)
		}
	}
}

func e07VerifyCodexMountBoundary(t *testing.T, mode string) {
	t.Helper()
	privateRoot := os.Getenv(e07PrivateRootEnv)
	if err := e07RequirePrivateCodexMount(privateRoot); err != nil {
		t.Fatal("private Codex mount boundary is incomplete")
	}
	namespace, err := os.Readlink("/proc/self/ns/mnt")
	if err != nil {
		t.Fatal("private mount namespace could not be verified")
	}
	pidNamespace, err := os.Readlink("/proc/self/ns/pid")
	if err != nil {
		t.Fatal("private PID namespace could not be verified")
	}
	if mode != "parent" && mode != "child" {
		t.Fatal("mount boundary test mode is invalid")
	}
	if mode == "child" && (namespace != os.Getenv(e07ExpectedMountNamespaceEnv) ||
		pidNamespace != os.Getenv(e07ExpectedPIDNamespaceEnv)) {
		t.Fatal("child process left the private mount or PID namespace")
	}
	globalCodex, err := e07AccountCodexHome()
	if err != nil {
		t.Fatal("account Codex state root could not be resolved")
	}
	if _, err := os.ReadDir(globalCodex); err != nil {
		t.Fatal("hidden global Codex mount could not be inspected")
	}
	probe := filepath.Join(globalCodex, ".e07-private-boundary-probe")
	err = os.WriteFile(probe, []byte("synthetic"), 0o600)
	if err == nil {
		_ = os.Remove(probe)
		t.Fatal("global Codex mount accepted a write")
	}
	if !errors.Is(err, syscall.EROFS) && !errors.Is(err, syscall.EACCES) {
		t.Fatal("global Codex write was rejected for an unexpected reason")
	}
	privateCodex := os.Getenv("CODEX_HOME")
	marker := "parent-state"
	if mode == "child" {
		marker = "child-state"
	}
	if err := os.WriteFile(filepath.Join(privateCodex, marker), []byte("synthetic"), 0o600); err != nil {
		t.Fatal("private Codex state did not accept a write")
	}
	if mode == "parent" {
		e07RunMountBoundaryChild(t, namespace)
	}
}

func e07RunMountBoundaryChild(t *testing.T, namespace string) {
	t.Helper()
	pidNamespace, err := os.Readlink("/proc/self/ns/pid")
	if err != nil {
		t.Fatal("private PID namespace could not be verified")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal("test executable could not be resolved")
	}
	command := exec.Command(executable, "-test.run=^TestE07CodexMountBoundaryHidesGlobalState$")
	command.Env = e07BoundaryChildEnvironment(os.Environ(), namespace, pidNamespace)
	if _, err := command.CombinedOutput(); err != nil {
		t.Fatal("Codex child-process namespace check failed")
	}
}

func e07BoundaryChildEnvironment(environment []string, namespace, pidNamespace string) []string {
	allowed := map[string]struct{}{
		"PATH": {}, "LANG": {}, "HOME": {}, "CODEX_HOME": {}, "CODEX_SQLITE_HOME": {},
		"TMPDIR": {}, "TEMP": {}, "TMP": {}, "XDG_CONFIG_HOME": {}, "XDG_DATA_HOME": {},
		"XDG_CACHE_HOME": {}, "XDG_STATE_HOME": {}, "XDG_RUNTIME_DIR": {}, e07PrivateRootEnv: {},
	}
	child := make([]string, 0, len(allowed)+2)
	for _, entry := range environment {
		name, _, found := strings.Cut(entry, "=")
		if _, isAllowed := allowed[name]; found && isAllowed {
			child = append(child, entry)
		}
	}
	return append(child, e07MountBoundaryModeEnv+"=child",
		e07ExpectedMountNamespaceEnv+"="+namespace, e07ExpectedPIDNamespaceEnv+"="+pidNamespace)
}

func e07AccountCodexHome() (string, error) {
	home, err := e07AccountHomeDirectory()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".codex")
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("account Codex state root is unavailable")
	}
	return path, nil
}
