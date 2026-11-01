//go:build linux

package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	active, err := e07CodexProcessActiveIn(processRoot, func(string) ([]byte, error) {
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
	active, err := e07CodexProcessActiveIn(processRoot, func(string) ([]byte, error) {
		return nil, os.ErrNotExist
	})
	if err != nil || active {
		t.Fatal("Codex process probe did not ignore an entry that exited during inspection")
	}
}

func TestE07CodexProcessUnsupportedPlatformFailsClosed(t *testing.T) {
	active, err := e07CodexProcessUnsupported()
	if err == nil || active {
		t.Fatal("unsupported process inspection was reported as no active Codex process")
	}
}

func e07CodexProcessActive() (bool, error) {
	return e07CodexProcessActiveIn("/proc", os.ReadFile)
}

func e07CodexProcessActiveIn(procRoot string, readProcessName func(string) ([]byte, error)) (bool, error) {
	if readProcessName == nil {
		return false, errors.New("active process names could not be inspected")
	}
	processes, err := os.ReadDir(procRoot)
	if err != nil {
		return false, errors.New("active Codex process state could not be inspected")
	}
	for _, process := range processes {
		if _, err := strconv.Atoi(process.Name()); err != nil {
			continue
		}
		name, err := readProcessName(filepath.Join(procRoot, process.Name(), "comm"))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return false, errors.New("active process names could not be inspected")
		}
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(string(name))), "codex") {
			return true, nil
		}
	}
	return false, nil
}
