package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func e07CopyTestExecutable(t *testing.T, source, directory string) string {
	t.Helper()
	input, err := os.Open(source)
	if err != nil {
		t.Fatal("test executable could not be opened for isolated mounting")
	}
	defer func() { _ = input.Close() }()

	target := filepath.Join(directory, "e07-test-binary")
	output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o700)
	if err != nil {
		t.Fatal("isolated test executable copy could not be created")
	}
	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		t.Fatal("isolated test executable copy could not be written")
	}
	if err := output.Close(); err != nil {
		t.Fatal("isolated test executable copy could not be closed")
	}
	return target
}

func e07CodexBoundaryArgs(bwrap, hiddenRoot, globalCodex, privateRoot, repository, executable string) []string {
	paths := map[string]string{
		"HOME": filepath.Join(e07PrivateMountRoot, "home"), "CODEX_HOME": filepath.Join(e07PrivateMountRoot, "codex-home"),
		"CODEX_SQLITE_HOME": filepath.Join(e07PrivateMountRoot, "codex-home"), "TMPDIR": filepath.Join(e07PrivateMountRoot, "tmp"),
		"TEMP": filepath.Join(e07PrivateMountRoot, "tmp"), "TMP": filepath.Join(e07PrivateMountRoot, "tmp"),
		"XDG_CONFIG_HOME": filepath.Join(e07PrivateMountRoot, "xdg", "config"), "XDG_DATA_HOME": filepath.Join(e07PrivateMountRoot, "xdg", "data"),
		"XDG_CACHE_HOME": filepath.Join(e07PrivateMountRoot, "xdg", "cache"), "XDG_STATE_HOME": filepath.Join(e07PrivateMountRoot, "xdg", "state"),
		"XDG_RUNTIME_DIR": filepath.Join(e07PrivateMountRoot, "xdg", "runtime"),
	}
	args := []string{
		"--unshare-pid", "--unshare-ipc", "--unshare-uts", "--ro-bind", "/", "/", "--dev", "/dev", "--proc", "/proc",
		"--tmpfs", "/run", "--dir", "/run/e07", "--dir", e07PrivateMountRoot,
		"--bind", privateRoot, e07PrivateMountRoot, "--ro-bind", hiddenRoot, globalCodex,
		"--setenv", "PATH", os.Getenv("PATH"), "--setenv", "LANG", "C.UTF-8",
		"--setenv", e07PrivateRootEnv, e07PrivateMountRoot,
	}
	for name, value := range paths {
		args = append(args, "--setenv", name, value)
	}
	args = append(args, "--setenv", e07MountBoundaryModeEnv, "parent", "--chdir", repository,
		"--", executable, "-test.run=^TestE07CodexMountBoundaryHidesGlobalState$")
	return append([]string{bwrap}, args...)
}

func e07RequirePrivateCodexMount(privateRoot string) error {
	if runtime.GOOS != "linux" || privateRoot == "" {
		return errors.New("private mount namespace is required")
	}
	root, err := filepath.Abs(privateRoot)
	if err != nil {
		return errors.New("private runtime root could not be resolved")
	}
	for _, name := range []string{"HOME", "CODEX_HOME", "CODEX_SQLITE_HOME", "TMPDIR", "TEMP", "TMP", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_STATE_HOME", "XDG_RUNTIME_DIR"} {
		path := os.Getenv(name)
		if path == "" || !e07PathWithinRoot(root, path) {
			return errors.New("Codex runtime path is outside the private root")
		}
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			return errors.New("Codex runtime directory is unavailable")
		}
	}
	globalCodex, err := e07AccountCodexHome()
	if err != nil {
		return err
	}
	mountInfo, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil || !e07MountIsReadOnly(mountInfo, globalCodex) {
		return errors.New("global Codex state is not hidden behind a read-only mount")
	}
	globalEntries, err := os.ReadDir(globalCodex)
	if err != nil || len(globalEntries) != 0 {
		return errors.New("global Codex state is visible inside the validation boundary")
	}
	repository, err := e07RepositoryRoot("")
	if err != nil {
		return errors.New("repository root could not be resolved")
	}
	if e07PathWithinRoot(repository, privateRoot) {
		return errors.New("private runtime mount is inside the repository")
	}
	repositoryReadOnly, repositoryMounted := e07MountReadOnlyForPath(mountInfo, repository)
	if !repositoryMounted || !repositoryReadOnly || !e07MountHasOption(mountInfo, privateRoot, "rw") {
		return errors.New("repository and private runtime mount permissions are invalid")
	}
	return nil
}

func e07SnapshotPrivateCodexState(codexHome string) ([]e07CodexTreeEntry, error) {
	directories := e07CodexRuntimeDirectories(codexHome)
	roots := make([]e07CodexTreeRoot, 0, len(directories))
	for index, directory := range directories {
		roots = append(roots, e07CodexTreeRoot{Name: "private_runtime_" + strconv.Itoa(index), Path: directory})
	}
	return e07SnapshotCodexTreeAt(roots)
}

func e07PrivateCodexStateSnapshotValid(entries []e07CodexTreeEntry, expectedRoots int) bool {
	return e07CodexTreeRootCount(entries) == expectedRoots
}

func e07CodexTreeRootCount(entries []e07CodexTreeEntry) int {
	roots := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		roots[entry.Root] = struct{}{}
	}
	return len(roots)
}

func e07PathWithinRoot(root, path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(root, abs)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func e07MountIsReadOnly(mountInfo []byte, target string) bool {
	return e07MountHasOption(mountInfo, target, "ro")
}

func e07MountHasOption(mountInfo []byte, target, option string) bool {
	target = filepath.Clean(target)
	for _, line := range strings.Split(string(mountInfo), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 || filepath.Clean(e07UnescapeMountField(fields[4])) != target {
			continue
		}
		for _, mountOption := range strings.Split(fields[5], ",") {
			if mountOption == option {
				return true
			}
		}
	}
	return false
}

func e07MountReadOnlyForPath(mountInfo []byte, target string) (bool, bool) {
	target = filepath.Clean(target)
	longest := -1
	readOnly := false
	found := false
	for _, line := range strings.Split(string(mountInfo), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		mountpoint := filepath.Clean(e07UnescapeMountField(fields[4]))
		if target != mountpoint && !strings.HasPrefix(target, strings.TrimRight(mountpoint, string(filepath.Separator))+string(filepath.Separator)) {
			continue
		}
		if len(mountpoint) <= longest {
			continue
		}
		longest = len(mountpoint)
		found = true
		readOnly = false
		for _, option := range strings.Split(fields[5], ",") {
			if option == "ro" {
				readOnly = true
				break
			}
		}
	}
	return readOnly, found
}

func e07UnescapeMountField(value string) string {
	return strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`).Replace(value)
}
