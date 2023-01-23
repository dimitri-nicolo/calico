// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package generator

import (
	"path/filepath"

	"github.com/projectcalico/calico/hack/cub-generator/pkg/fs"
)

// Project contains metadata about the project we want to generate and how we want to generate it
type Project struct {
	Path string
	Name string

	gen Generator
}

// Generator defines how a project will be generated
type Generator interface {
	Render(p *Project, workingDir string) error
}

// NewProject creates a new Project by providing its metadata and generator
func NewProject(location string, name string, gen Generator) *Project {
	return &Project{Path: location, Name: name, gen: gen}
}

// Render will generate the structure of the project or return an error in case of failure
func (p *Project) Render() error {
	var workingDir = filepath.Join(p.Path, p.Name)
	if err := fs.CreateDir(workingDir); err != nil {
		return err
	}
	if err := p.gen.Render(p, workingDir); err != nil {
		return err
	}

	return nil
}
