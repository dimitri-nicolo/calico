// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package fs

import (
	"fmt"
	"os"
	"path/filepath"
)

// CreateDir creates a directory with the given path
func CreateDir(path string) error {
	fmt.Printf("Creating %s directory \n", path)
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

// CreateFile creates a file with the given path
func CreateFile(name string, path string) (*os.File, error) {
	var fileName = filepath.Join(path, name)
	fmt.Printf("Creating %s in %s \n", name, path)
	return os.Create(fileName)
}

// CloseFile closes a given file
func CloseFile(file *os.File) {
	func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Errorf("failed to close File %s", file.Name())
		}
	}(file)
}
