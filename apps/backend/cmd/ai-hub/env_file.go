package main

import (
	"os"
	"path/filepath"
)

func defaultEnvFilePath() string {
	workingDir, err := os.Getwd()
	if err != nil {
		return ".env"
	}

	return defaultEnvFilePathFrom(workingDir)
}

func defaultEnvFilePathFrom(startDir string) string {
	moduleDir, err := filepath.Abs(startDir)
	if err != nil {
		return filepath.Join(startDir, ".env")
	}

	workingDirEnvPath := filepath.Join(moduleDir, ".env")
	for dir := moduleDir; ; dir = filepath.Dir(dir) {
		moduleFile := filepath.Join(dir, "go.mod")
		if info, err := os.Stat(moduleFile); err == nil && !info.IsDir() {
			return filepath.Join(dir, ".env")
		}

		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			return workingDirEnvPath
		}
	}
}
