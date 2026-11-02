package support

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func E07IsSecretEnvironmentFile(name string) bool {
	filename := filepath.Base(name)
	return filename == ".env" ||
		(strings.HasPrefix(filename, ".env.") && !strings.HasSuffix(filename, ".example"))
}

func E07IsSecretEnvironmentEntry(path, name string, info os.FileInfo) bool {
	if info.IsDir() {
		return false
	}
	if E07IsSecretEnvironmentFile(name) || E07IsSecretEnvironmentFile(path) {
		return true
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	target, err := os.Readlink(path)
	return err == nil && E07IsSecretEnvironmentFile(target)
}

func E07DigestSecretEnvironmentMetadata(hasher io.Writer, name string, info os.FileInfo) error {
	if _, err := fmt.Fprintf(hasher, "secret-env:%s:%s\n", name, E07StableFileMetadata(info)); err != nil {
		return errors.New("environment metadata fingerprint could not be written")
	}
	return nil
}
