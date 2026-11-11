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
		workspaceFile := filepath.Join(dir, "pnpm-workspace.yaml")
		if info, err := os.Stat(workspaceFile); err == nil && !info.IsDir() {
			return filepath.Join(dir, ".env")
		}

		moduleFile := filepath.Join(dir, "go.mod")
		if info, err := os.Stat(moduleFile); err == nil && !info.IsDir() {
			localEnv := filepath.Join(dir, ".env")
			if envInfo, envErr := os.Stat(localEnv); envErr == nil && !envInfo.IsDir() {
				return localEnv
			}
			if workspaceRoot := findWorkspaceRoot(filepath.Dir(dir)); workspaceRoot != "" {
				return filepath.Join(workspaceRoot, ".env")
			}

			return localEnv
		}

		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			return workingDirEnvPath
		}
	}
}

func findWorkspaceRoot(startDir string) string {
	for dir := startDir; ; dir = filepath.Dir(dir) {
		workspaceFile := filepath.Join(dir, "pnpm-workspace.yaml")
		if info, err := os.Stat(workspaceFile); err == nil && !info.IsDir() {
			return dir
		}

		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			return ""
		}
	}
}
