//go:build linux

package codex

import (
	e07support "aihub/cmd/ai-hub/e07/support"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"unsafe"
)

func TestE07CodexProcessNameHelper(t *testing.T) {
	if os.Getenv("E07_CODEX_PROCESS_HELPER") != "1" {
		return
	}
	name := [16]byte{'c', 'o', 'd', 'e', 'x', '-', 'e', '0', '7', '-', 'p', 'r', 'o', 'b', 'e'}
	_, _, errno := syscall.RawSyscall6(syscall.SYS_PRCTL, 15, uintptr(unsafe.Pointer(&name[0])), 0, 0, 0, 0)
	if errno != 0 {
		return
	}
	_, _ = os.Stdout.Write([]byte{1})
	_, _ = io.Copy(io.Discard, os.Stdin)
}

func TestE07CodexProcessInspectionFailsClosedOnPermissionError(t *testing.T) {
	processRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(processRoot, "123"), 0o700); err != nil {
		t.Fatal("synthetic process entry could not be created")
	}
	active, err := e07support.E07CodexProcessActiveIn(processRoot, func(string) ([]byte, error) {
		return nil, os.ErrPermission
	})
	if err == nil || active {
		t.Fatal("Codex process probe treated an unreadable process name as inactive")
	}
}

func TestE07CodexProcessInspectionIgnoresVanishedProcess(t *testing.T) {
	processRoot := t.TempDir()
	if err := os.Mkdir(filepath.Join(processRoot, "123"), 0o700); err != nil {
		t.Fatal("synthetic process entry could not be created")
	}
	active, err := e07support.E07CodexProcessActiveIn(processRoot, func(string) ([]byte, error) {
		return nil, os.ErrNotExist
	})
	if err != nil || active {
		t.Fatal("Codex process probe did not ignore an entry that exited during inspection")
	}
}

func TestE07CodexProcessUnsupportedPlatformFailsClosed(t *testing.T) {
	active, err := e07support.E07CodexProcessUnsupported()
	if err == nil || active {
		t.Fatal("unsupported process inspection was reported as no active Codex process")
	}
}
