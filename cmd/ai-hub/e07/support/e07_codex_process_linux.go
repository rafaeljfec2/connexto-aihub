//go:build linux

package support

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func E07CodexProcessActive() (bool, error) {
	return E07CodexProcessActiveIn("/proc", os.ReadFile)
}

func E07CodexProcessActiveIn(procRoot string, readProcessName func(string) ([]byte, error)) (bool, error) {
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
