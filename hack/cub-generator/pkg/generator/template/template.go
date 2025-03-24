// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package template

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// File represent a template that will be used to render a project
type File struct {
	Name string
	Path string

	Template *template.Template
}

// Templates is a collection of file templates
type Templates []File

// Len returns the length of the collection of file templates
func (t Templates) Len() int {
	return len(t)
}

// Less compares two file templates
func (t Templates) Less(i, j int) bool {
	return filepath.Join(t[i].Path, t[i].Name) > filepath.Join(t[j].Path, t[j].Name)
}

// Swap swaps the position of two file templates
func (t Templates) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

// LoadTemplates load templates from file system to memory in order
// to render a project. Templates must be placed under baseDirName
// embedded directory
func LoadTemplates(templates fs.FS, baseDirName string) (Templates, error) {
	if baseDirName == "" {
		return nil, fmt.Errorf("baseDirName cannot be empty string")
	}

	var templateFiles []File

	var err = fs.WalkDir(
		templates,
		".",
		func(path string, d fs.DirEntry, err error) error {
			if !d.IsDir() {
				var name = filepath.Base(path)
				tpl, err := template.New(name).
					Funcs(sprig.TxtFuncMap()).
					ParseFS(templates, path)

				if err != nil {
					return err
				}

				if !strings.Contains(path, baseDirName) {
					return fmt.Errorf("Malformed template path")
				}

				appPath, err := filepath.Rel(baseDirName, path)

				if err != nil {
					return err
				}

				file := File{
					Path: filepath.Dir(appPath),
					Name: name,

					Template: tpl,
				}
				templateFiles = append(templateFiles, file)
			}

			return nil
		},
	)

	return templateFiles, err
}
