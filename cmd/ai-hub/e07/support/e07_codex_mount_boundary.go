package support

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

const (
	E07MountBoundaryModeEnv      = "AIHUB_E07_MOUNT_BOUNDARY_MODE"
	E07PrivateRootEnv            = "AIHUB_E07_PRIVATE_ROOT"
	E07ExpectedMountNamespaceEnv = "AIHUB_E07_EXPECTED_MOUNT_NAMESPACE"
	E07ExpectedPIDNamespaceEnv   = "AIHUB_E07_EXPECTED_PID_NAMESPACE"
	E07PrivateMountRoot          = "/run/e07/private"
)

func E07VerifyCodexMountBoundary(t *testing.T, mode string) {
	t.Helper()
	privateRoot := os.Getenv(E07PrivateRootEnv)
	if err := E07RequirePrivateCodexMount(privateRoot); err != nil {
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
	if mode == "child" && (namespace != os.Getenv(E07ExpectedMountNamespaceEnv) ||
		pidNamespace != os.Getenv(E07ExpectedPIDNamespaceEnv)) {
		t.Fatal("child process left the private mount or PID namespace")
	}
	globalCodex, err := E07AccountCodexHome()
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
		E07RunMountBoundaryChild(t, namespace)
	}
}

func E07RunMountBoundaryChild(t *testing.T, namespace string) {
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
	command.Env = E07BoundaryChildEnvironment(os.Environ(), namespace, pidNamespace)
	if _, err := command.CombinedOutput(); err != nil {
		t.Fatal("Codex child-process namespace check failed")
	}
}

func E07BoundaryChildEnvironment(environment []string, namespace, pidNamespace string) []string {
	allowed := map[string]struct{}{
		"PATH": {}, "LANG": {}, "HOME": {}, "CODEX_HOME": {}, "CODEX_SQLITE_HOME": {},
		"TMPDIR": {}, "TEMP": {}, "TMP": {}, "XDG_CONFIG_HOME": {}, "XDG_DATA_HOME": {},
		"XDG_CACHE_HOME": {}, "XDG_STATE_HOME": {}, "XDG_RUNTIME_DIR": {}, E07PrivateRootEnv: {},
	}
	child := make([]string, 0, len(allowed)+2)
	for _, entry := range environment {
		name, _, found := strings.Cut(entry, "=")
		if _, isAllowed := allowed[name]; found && isAllowed {
			child = append(child, entry)
		}
	}
	return append(child, E07MountBoundaryModeEnv+"=child",
		E07ExpectedMountNamespaceEnv+"="+namespace, E07ExpectedPIDNamespaceEnv+"="+pidNamespace)
}

func E07AccountCodexHome() (string, error) {
	home, err := E07AccountHomeDirectory()
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
