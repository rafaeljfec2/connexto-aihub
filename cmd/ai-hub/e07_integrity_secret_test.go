package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func e07IsSecretEnvironmentFile(name string) bool {
	filename := filepath.Base(name)
	return filename == ".env" ||
		(strings.HasPrefix(filename, ".env.") && !strings.HasSuffix(filename, ".example"))
}

func e07IsSecretEnvironmentEntry(path, name string, info os.FileInfo) bool {
	if info.IsDir() {
		return false
	}
	if e07IsSecretEnvironmentFile(name) || e07IsSecretEnvironmentFile(path) {
		return true
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	target, err := os.Readlink(path)
	return err == nil && e07IsSecretEnvironmentFile(target)
}

func e07DigestSecretEnvironmentMetadata(hasher io.Writer, name string, info os.FileInfo) error {
	if _, err := fmt.Fprintf(hasher, "secret-env:%s:%s\n", name, e07StableFileMetadata(info)); err != nil {
		return errors.New("environment metadata fingerprint could not be written")
	}
	return nil
}

func TestE07SecretEnvironmentChangesAreDetectableWithoutReadingContents(t *testing.T) {
	root := e07IntegrityFixture(t)
	files := map[string]string{
		".env.production":        "PROVIDER_KEY=synthetic-one\n",
		"nested/.env":            "GATEWAY_KEY=synthetic-one\n",
		"nested/.env.production": "CLIENT_KEY=synthetic-one\n",
		".env.example":           "EXAMPLE_KEY=placeholder\n",
	}
	for relative, content := range files {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal("environment fixture directory could not be created")
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal("environment fixture could not be written")
		}
	}

	before, err := e07SnapshotRepository(root)
	if err != nil {
		t.Fatalf("initial integrity snapshot failed: %v", err)
	}
	for relative, content := range map[string]string{
		".env":                   "ROOT_KEY=synthetic-content-with-a-different-length\n",
		".env.production":        "PROVIDER_KEY=synthetic-two-and-different-length\n",
		"nested/.env":            "GATEWAY_KEY=synthetic-two-with-more-bytes\n",
		"nested/.env.production": "CLIENT_KEY=synthetic-two-with-more-bytes\n",
		"nested/.env.local":      "NESTED_KEY=synthetic-two-with-more-bytes\n",
	} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(relative)), []byte(content), 0o600); err != nil {
			t.Fatal("synthetic environment content could not be changed")
		}
	}
	after, err := e07SnapshotRepository(root)
	if err != nil {
		t.Fatalf("updated integrity snapshot failed: %v", err)
	}
	if before.Tree == after.Tree || before.Env == after.Env || before.Tracked == after.Tracked {
		t.Fatal("secret environment file changes were not detected from filesystem metadata")
	}

	if err := os.WriteFile(filepath.Join(root, ".env.example"), []byte("EXAMPLE_KEY=changed-template\n"), 0o600); err != nil {
		t.Fatal("environment template could not be changed")
	}
	templateChanged, err := e07FingerprintTree(root)
	if err != nil {
		t.Fatalf("template integrity fingerprint failed: %v", err)
	}
	if templateChanged == after.Tree {
		t.Fatal("non-secret .env.example content was excluded from the tree fingerprint")
	}
}

func TestE07SecretEnvironmentFilesAreNotOpenedForFingerprinting(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-denied read guard requires a non-root test process")
	}
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	if err := os.WriteFile(path, []byte("KEY=synthetic-value\n"), 0o600); err != nil {
		t.Fatal("synthetic environment file could not be created")
	}
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal("synthetic environment file could not be protected")
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })

	if _, err := e07FingerprintTree(root); err != nil {
		t.Fatal("tree fingerprint attempted to read a protected environment file")
	}
	files, err := e07EnvFiles(root)
	if err != nil {
		t.Fatalf("environment metadata discovery failed: %v", err)
	}
	if _, err := e07FingerprintEnvFiles(root, files); err != nil {
		t.Fatal("environment fingerprint attempted to read a protected environment file")
	}
}

func TestE07SecretEnvironmentFingerprintTracksPermissionsOnly(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env.production")
	if err := os.WriteFile(path, []byte("KEY=synthetic-value\n"), 0o600); err != nil {
		t.Fatal("synthetic environment file could not be created")
	}
	files, err := e07EnvFiles(root)
	if err != nil {
		t.Fatalf("environment metadata discovery failed: %v", err)
	}
	before, err := e07FingerprintEnvFiles(root, files)
	if err != nil {
		t.Fatalf("initial environment metadata fingerprint failed: %v", err)
	}
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal("synthetic environment permissions could not be changed")
	}
	after, err := e07FingerprintEnvFiles(root, files)
	if err != nil {
		t.Fatalf("updated environment metadata fingerprint failed: %v", err)
	}
	if before == after {
		t.Fatal("environment permission changes were not represented in metadata")
	}
}

func TestE07GitStateExcludesSecretEnvironmentPaths(t *testing.T) {
	root := e07IntegrityFixture(t)
	before, err := e07GitState(root)
	if err != nil {
		t.Fatalf("initial Git-state snapshot failed: %v", err)
	}
	for relative, content := range map[string]string{
		".env":              "ROOT_KEY=synthetic-value-with-a-different-length\n",
		"nested/.env.local": "NESTED_KEY=synthetic-value-with-a-different-length\n",
	} {
		if err := os.WriteFile(filepath.Join(root, relative), []byte(content), 0o600); err != nil {
			t.Fatal("synthetic environment content could not be changed")
		}
	}
	afterSecrets, err := e07GitState(root)
	if err != nil {
		t.Fatalf("Git-state snapshot after dotenv changes failed: %v", err)
	}
	if before != afterSecrets {
		t.Fatal("Git-state fingerprint included secret environment paths")
	}

	if err := os.WriteFile(filepath.Join(root, "new_source.go"), []byte("package fixture\n"), 0o600); err != nil {
		t.Fatal("untracked source fixture could not be created")
	}
	afterSource, err := e07GitState(root)
	if err != nil {
		t.Fatalf("Git-state snapshot after source change failed: %v", err)
	}
	if afterSecrets == afterSource {
		t.Fatal("Git-state fingerprint ignored a tracked source change")
	}
}

func TestE07SecretSymlinkTargetChangesAreDetectableWithoutReadingContents(t *testing.T) {
	root := t.TempDir()
	targetRoot := t.TempDir()
	target := filepath.Join(targetRoot, ".env.production")
	if err := os.WriteFile(target, []byte("KEY=synthetic-one\n"), 0o600); err != nil {
		t.Fatal("synthetic symlink target could not be created")
	}
	link := filepath.Join(root, "credential-link")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symbolic links are unavailable")
	}

	before, err := e07FingerprintTree(root)
	if err != nil {
		t.Fatalf("initial symlink fingerprint failed: %v", err)
	}
	if err := os.WriteFile(target, []byte("KEY=synthetic-content-with-a-different-length\n"), 0o600); err != nil {
		t.Fatal("synthetic symlink target content could not be changed")
	}
	after, err := e07FingerprintTree(root)
	if err != nil {
		t.Fatalf("updated symlink fingerprint failed: %v", err)
	}
	if before == after {
		t.Fatal("secret symlink target change was not detected from filesystem metadata")
	}
}

func TestE07SecretSymlinkDoesNotOpenProtectedTarget(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-denied read guard requires a non-root test process")
	}
	root := t.TempDir()
	targetRoot := t.TempDir()
	target := filepath.Join(targetRoot, ".env.local")
	aliasTarget := filepath.Join(targetRoot, "credentials.dat")
	if err := os.WriteFile(target, []byte("KEY=synthetic-value\n"), 0o600); err != nil {
		t.Fatal("synthetic environment target could not be created")
	}
	if err := os.WriteFile(aliasTarget, []byte("KEY=synthetic-value\n"), 0o600); err != nil {
		t.Fatal("synthetic dotenv alias target could not be created")
	}
	if err := os.Chmod(target, 0); err != nil {
		t.Fatal("synthetic environment target could not be protected")
	}
	if err := os.Chmod(aliasTarget, 0); err != nil {
		t.Fatal("synthetic dotenv alias target could not be protected")
	}
	t.Cleanup(func() {
		_ = os.Chmod(target, 0o600)
		_ = os.Chmod(aliasTarget, 0o600)
	})
	if err := os.Symlink(target, filepath.Join(root, "environment-link")); err != nil {
		t.Skip("symbolic links are unavailable")
	}
	if err := os.Symlink(aliasTarget, filepath.Join(root, ".env")); err != nil {
		t.Fatal("synthetic dotenv alias could not be created")
	}
	if _, err := e07FingerprintTree(root); err != nil {
		t.Fatal("tree fingerprint attempted to open a protected dotenv symlink target")
	}
}
