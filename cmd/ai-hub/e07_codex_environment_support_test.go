package main

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func e07CodexProcessUnsupported() (bool, error) {
	return false, errors.New("Codex process detection is unsupported on this platform")
}

func e07CodexRuntimeDirectories(codexHome string) []string {
	root := filepath.Dir(codexHome)
	return []string{
		codexHome,
		filepath.Join(root, "tmp"),
		filepath.Join(root, "xdg", "config"),
		filepath.Join(root, "xdg", "data"),
		filepath.Join(root, "xdg", "cache"),
		filepath.Join(root, "xdg", "state"),
		filepath.Join(root, "xdg", "runtime"),
	}
}

func e07CodexRuntimeEnvironment(codexHome string) []string {
	root := filepath.Dir(codexHome)
	values := map[string]string{
		"HOME": codexHome, "CODEX_HOME": codexHome,
		"CODEX_SQLITE_HOME": codexHome,
		"TMPDIR":            filepath.Join(root, "tmp"), "TEMP": filepath.Join(root, "tmp"), "TMP": filepath.Join(root, "tmp"),
		"XDG_CONFIG_HOME": filepath.Join(root, "xdg", "config"),
		"XDG_DATA_HOME":   filepath.Join(root, "xdg", "data"),
		"XDG_CACHE_HOME":  filepath.Join(root, "xdg", "cache"),
		"XDG_STATE_HOME":  filepath.Join(root, "xdg", "state"),
		"XDG_RUNTIME_DIR": filepath.Join(root, "xdg", "runtime"),
	}
	result := make([]string, 0, len(values))
	for name, value := range values {
		result = append(result, name+"="+value)
	}
	return result
}

func e07CodexShellEnvironment(environment []string) ([]string, error) {
	values := make(map[string]string, len(environment))
	for _, entry := range environment {
		name, value, found := strings.Cut(entry, "=")
		if found {
			values[name] = value
		}
	}
	allowed := []string{"PATH", "HOME", "TMPDIR"}
	if runtime.GOOS == "windows" {
		allowed = append(allowed, "TEMP", "TMP", "SYSTEMROOT", "WINDIR")
	}
	if values["PATH"] == "" || values["HOME"] == "" || values["TMPDIR"] == "" {
		return nil, errors.New("Codex shell environment is missing a required temporary runtime value")
	}
	result := make([]string, 0, len(allowed))
	for _, name := range allowed {
		if value, found := values[name]; found {
			result = append(result, name+"="+value)
		}
	}
	return result, nil
}

func e07EnsureCodexRuntimeDirectories(codexHome string) error {
	for _, directory := range e07CodexRuntimeDirectories(codexHome) {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return errors.New("isolated Codex runtime directory could not be created")
		}
		if err := os.Chmod(directory, 0o700); err != nil {
			return errors.New("isolated Codex runtime directory could not be protected")
		}
	}
	return nil
}
