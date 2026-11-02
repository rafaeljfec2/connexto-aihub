package support

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// e07FingerprintTree fingerprints every entry below a path. Symbolic links
// record their target and, when resolvable, the target fingerprint. Secret
// dotenv entries use metadata so their contents are never opened. Directory
// links are followed with cycle protection.
func E07FingerprintTree(path string) (string, error) {
	hasher := sha256.New()
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		_, _ = io.WriteString(hasher, "absent")

		return hex.EncodeToString(hasher.Sum(nil)), nil
	}
	if err != nil {
		return "", errors.New("integrity fingerprint could not inspect a path")
	}
	entries := 0
	if err := E07DigestPath(hasher, path, "", info, make(map[string]struct{}), &entries); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func E07DigestPath(hasher io.Writer, path, name string, info os.FileInfo, visiting map[string]struct{}, entries *int) error {
	*entries++
	if *entries > E07IntegrityMaxEntries {
		return errors.New("integrity fingerprint exceeded the validation bound")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return E07DigestSymlink(hasher, path, name, info, visiting, entries)
	}
	if E07IsSecretEnvironmentEntry(path, name, info) {
		return E07DigestSecretEnvironmentMetadata(hasher, name, info)
	}
	if info.IsDir() {
		return E07DigestDirectory(hasher, path, name, info, visiting, entries)
	}
	if !info.Mode().IsRegular() {
		_, _ = fmt.Fprintf(hasher, "special:%s:%o:%d\n", name, info.Mode(), info.Size())

		return nil
	}

	return E07HashFile(hasher, name, path, info)
}

func E07DigestDirectory(hasher io.Writer, path, name string, info os.FileInfo, visiting map[string]struct{}, entries *int) error {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return errors.New("integrity fingerprint could not resolve a directory")
	}
	if _, cycle := visiting[resolved]; cycle {
		_, _ = fmt.Fprintf(hasher, "cycle:%s\n", name)

		return nil
	}
	visiting[resolved] = struct{}{}
	defer delete(visiting, resolved)

	_, _ = fmt.Fprintf(hasher, "dir:%s:%o\n", name, info.Mode().Perm())
	children, err := os.ReadDir(path)
	if err != nil {
		return errors.New("integrity fingerprint could not traverse a path")
	}
	for _, child := range children {
		if child.IsDir() && child.Name() == ".git" {
			continue
		}
		childPath := filepath.Join(path, child.Name())
		childName := child.Name()
		if name != "" {
			childName = name + "/" + child.Name()
		}
		childInfo, err := os.Lstat(childPath)
		if err != nil {
			return errors.New("integrity fingerprint could not inspect an entry")
		}
		if err := E07DigestPath(hasher, childPath, childName, childInfo, visiting, entries); err != nil {
			return err
		}
	}

	return nil
}

func E07DigestSymlink(hasher io.Writer, path, name string, info os.FileInfo, visiting map[string]struct{}, entries *int) error {
	target, err := os.Readlink(path)
	if err != nil {
		return errors.New("integrity fingerprint could not inspect a symbolic link")
	}
	_, _ = fmt.Fprintf(hasher, "link:%s:%o:%s\n", name, info.Mode().Perm(), target)

	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		_, _ = fmt.Fprintf(hasher, "link-unresolved:%s\n", name)

		return nil
	}
	resolvedInfo, err := os.Stat(resolved)
	if err != nil {
		_, _ = fmt.Fprintf(hasher, "link-unreadable:%s\n", name)

		return nil
	}
	if E07IsSecretEnvironmentEntry(path, name, info) ||
		E07IsSecretEnvironmentEntry(resolved, name+"->", resolvedInfo) {
		if err := E07DigestSecretEnvironmentMetadata(hasher, name, info); err != nil {
			return err
		}
		return E07DigestSecretEnvironmentMetadata(hasher, name+"->", resolvedInfo)
	}

	return E07DigestPath(hasher, resolved, name+"->", resolvedInfo, visiting, entries)
}

// e07ErrorEnvelopePresent classifies a generic upstream error envelope without
// retaining its content.
func E07ErrorEnvelopePresent(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	var envelope struct {
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return false
	}
	value := bytes.TrimSpace(envelope.Error)

	return len(value) > 0 && !bytes.Equal(value, []byte("null"))
}
