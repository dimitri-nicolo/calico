package main

import (
	"io/ioutil"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
)

func traverseDir(dir string, skipSubDir bool, substr string, forEach func(string) error) error {
	clog := log.WithField("dir", dir)
	clog.Debug("Traversing")

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		clog.WithError(err).Error("failed to read dir")
		return err
	}

	for _, file := range files {
		clog2 := clog.WithField("file", file.Name())

		// Conditionally skip subdirectories.
		if skipSubDir && file.IsDir() {
			clog2.Debug("skipping directory")
			continue
		}

		// Conditionally apply substring filter.
		fileName := file.Name()
		if substr != "" && strings.Contains(fileName, substr) {
			clog2.WithField("substr", substr).Debug("skipping file containing substr")
		}

		// Invoke the function to call for each traversal.
		if err := forEach(path.Join(dir, file.Name())); err != nil {
			clog2.WithError(err).Error("error occurred while traversing file")
			return err
		}
	}

	return nil
}
