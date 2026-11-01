package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"aihub/internal/config"
)

const e07ReadOnlyMountChild = "AIHUB_E07_READ_ONLY_MOUNT_CHILD"

func TestE07ReadOnlyMountLoadsDotEnvAndRejectsWrites(t *testing.T) {
	if os.Getenv(e07ReadOnlyMountChild) == "1" {
		e07CheckReadOnlyMount(t)
		return
	}
	if runtime.GOOS != "linux" {
		t.Skip("read-only bubblewrap mount test requires Linux")
	}
	bwrap, err := exec.LookPath("bwrap")
	if err != nil {
		t.Skip("bubblewrap is unavailable")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("E07_SYNTHETIC_KEY=synthetic-value\n"), 0o600); err != nil {
		t.Fatal("synthetic dotenv fixture could not be written")
	}
	if err := os.WriteFile(filepath.Join(root, "source.go"), []byte("package fixture\n"), 0o600); err != nil {
		t.Fatal("source fixture could not be written")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal("test executable could not be resolved")
	}
	e07CopyTestExecutable(t, executable, root)
	command := exec.Command(bwrap,
		"--ro-bind", "/", "/",
		"--tmpfs", "/tmp",
		"--dir", "/tmp/e07",
		"--ro-bind", filepath.Join(root, "e07-test-binary"), "/tmp/e07/test-binary",
		"--dev", "/dev",
		"--proc", "/proc",
		"--tmpfs", "/run",
		"--dir", "/run/e07-test",
		"--dir", "/run/e07-test/repository",
		"--ro-bind", root, "/run/e07-test/repository",
		"--chdir", "/run/e07-test/repository",
		"--dir", "/run/e07-test/home",
		"--setenv", "HOME", "/run/e07-test/home",
		"--setenv", "TMPDIR", "/run/e07-test",
		"--setenv", e07ReadOnlyMountChild, "1",
		"--", "/tmp/e07/test-binary", "-test.run=^TestE07ReadOnlyMountLoadsDotEnvAndRejectsWrites$")
	command.Env = []string{"PATH=" + os.Getenv("PATH")}
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("read-only mount behavior could not be verified: %v: %s", err, output)
	}
}

func e07CheckReadOnlyMount(t *testing.T) {
	t.Helper()
	if err := config.LoadDotEnv(".env"); err != nil {
		t.Fatal("production dotenv parser could not read the read-only fixture")
	}
	if os.Getenv("E07_SYNTHETIC_KEY") != "synthetic-value" {
		t.Fatal("production dotenv parser did not load the synthetic fixture")
	}
	for _, path := range []string{".env", "source.go"} {
		err := os.WriteFile(path, []byte("overwritten"), 0o600)
		if err == nil || (!errors.Is(err, syscall.EROFS) && !errors.Is(err, syscall.EACCES)) {
			t.Fatal("read-only repository mount accepted a write")
		}
	}
}
