// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package app

import (
	"embed"
	"os"
	"path/filepath"
	"strings"

	projFs "github.com/projectcalico/calico/hack/cub-generator/pkg/fs"
	"github.com/projectcalico/calico/hack/cub-generator/pkg/generator"
	"github.com/projectcalico/calico/hack/cub-generator/pkg/generator/template"
)

// AppTemplates are embedded template file to generate a project
//
//go:embed template/*
var AppTemplates embed.FS

type application struct {
	templates []template.File
}

// NewApp generates a project for a go application
func NewApp(templateFiles []template.File) generator.Generator {
	return &application{
		templates: templateFiles,
	}
}

func render(f template.File, p *generator.Project, workingDir string) error {

	var err error
	var file *os.File

	var path = filepath.Join(workingDir, f.Path)
	if err := projFs.CreateDir(path); err != nil {
		return err
	}
	var name = strings.TrimSuffix(f.Name, ".template")
	file, err = projFs.CreateFile(name, path)
	defer projFs.CloseFile(file)

	if err != nil {
		return err
	}

	if err := f.Template.Execute(file, p); err != nil {
		return err
	}
	return nil
}

// Render will render a project according to its templates. It will create a matching
// directory structure and insert the project name where the template requires it
func (app *application) Render(p *generator.Project, workingDir string) error {
	for _, f := range app.templates {
		if err := render(f, p, workingDir); err != nil {
			return err
		}
	}

	return nil
}
