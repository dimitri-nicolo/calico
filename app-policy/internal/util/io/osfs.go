package io

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// dirFS implements os.DirFS as fs.FS using methods on os to read from the system.
// Note that this implementation is not a compliant fs.FS, as they should only
// accept posix-style, relative paths, but as this is an internal implementation
// detail, we get the abstraction we need while being able to handle paths as
// the os package otherwise would.
// More context in: https://github.com/golang/go/issues/44279
// inspiration from the following:
// - https://github.com/corazawaf/coraza/blob/main/internal/io/file.go
// - https://github.com/jcchavezs/mergefs/blob/main/io/os.go
// - libexec/src/os/file.go
var (
	_ fs.FS = dirFS("")
)

type dirFS string

func DirFS(dir string) fs.FS {
	return dirFS(dir)
}

func (dir dirFS) Open(name string) (fs.File, error) {
	logrus.Debug("Opening file: ", filepath.Join(string(dir), name))
	return os.Open(filepath.Join(string(dir), name))
}

func (dir dirFS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(string(dir), name))
}

func (dir dirFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(filepath.Join(string(dir), name))
}

func (dir dirFS) Glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}
