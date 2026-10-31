package records

import (
	"io"
	"os"
)

const (
	directoryMode = 0o700
	fileMode      = 0o600
)

// appendFile is the minimal mutation surface used by the writer.
type appendFile interface {
	io.Writer
	io.Closer
	Sync() error
	Truncate(size int64) error
	Stat() (os.FileInfo, error)
}

// fileOps isolates filesystem effects so tests can inject failures.
type fileOps interface {
	mkdirAll(path string, perm os.FileMode) error
	chmod(path string, perm os.FileMode) error
	openAppend(path string) (appendFile, error)
	openExclusive(path string) (appendFile, error)
	openTemp(path string) (appendFile, error)
	openRead(path string) (io.ReadCloser, error)
	rename(oldPath, newPath string) error
	remove(path string) error
	syncDir(dir string) error
	readDir(path string) ([]os.DirEntry, error)
}

// osFileOps is the production filesystem implementation.
type osFileOps struct{}

func (osFileOps) mkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (osFileOps) chmod(path string, perm os.FileMode) error {
	return os.Chmod(path, perm)
}

func (osFileOps) openAppend(path string) (appendFile, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, fileMode)
}

func (osFileOps) openExclusive(path string) (appendFile, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fileMode)
}

func (osFileOps) openTemp(path string) (appendFile, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fileMode)
}

func (osFileOps) openRead(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

func (osFileOps) rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

func (osFileOps) remove(path string) error {
	return os.Remove(path)
}

func (osFileOps) syncDir(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = handle.Close() }()

	return handle.Sync()
}

func (osFileOps) readDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}
