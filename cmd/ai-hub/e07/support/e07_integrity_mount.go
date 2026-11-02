package support

import (
	"errors"
	"os"
	"syscall"
	"testing"

	"aihub/internal/config"
)

const E07ReadOnlyMountChild = "AIHUB_E07_READ_ONLY_MOUNT_CHILD"

func E07CheckReadOnlyMount(t *testing.T) {
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
