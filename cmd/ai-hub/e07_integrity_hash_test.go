package main

import (
	"errors"
	"fmt"
	"io"
	"os"
)

func e07HashFile(hasher io.Writer, name, path string, info os.FileInfo) error {
	file, err := os.Open(path)
	if err != nil {
		return errors.New("integrity fingerprint could not read a file")
	}
	defer func() { _ = file.Close() }()
	_, _ = fmt.Fprintf(hasher, "file:%s:%o:%d:", name, info.Mode().Perm(), info.Size())
	if _, err := io.Copy(hasher, file); err != nil {
		return errors.New("integrity fingerprint could not hash a file")
	}
	_, _ = io.WriteString(hasher, "\n")
	return nil
}
