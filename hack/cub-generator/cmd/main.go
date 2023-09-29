// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package main

import (
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/projectcalico/calico/hack/cub-generator/pkg/generator"
	"github.com/projectcalico/calico/hack/cub-generator/pkg/generator/app"
	"github.com/projectcalico/calico/hack/cub-generator/pkg/generator/template"
	"github.com/projectcalico/calico/hack/cub-generator/pkg/version"
)

type Options struct {
	// Name of the newly generated project
	Name string `short:"n" long:"name" description:"Project name" required:"true"`

	// Location of the newly generated project
	Location string `short:"l" long:"location" description:"Path of the project" default:"."`

	// Version of the project generator
	Version func() `long:"version" description:"Version for cub-generator"`

	// Type will generate a number of different project templates
	Type string `long:"type" choice:"http-server" choice:"skeleton" choice:"proxy"`
}

var options Options

var parser = flags.NewParser(&options, flags.Default)

func main() {
	options.Version = func() {
		version.Version()
		os.Exit(0)
	}

	if _, err := parser.Parse(); err != nil {
		switch flagsErr := err.(type) {
		case flags.ErrorType:
			if flagsErr == flags.ErrHelp {
				os.Exit(0)
			}
			os.Exit(1)
		default:
			os.Exit(1)
		}
	}

	var projectType = app.SkeletonTemplates
	switch options.Type {
	case "http-server":
		projectType = app.GoServerTemplates
	case "proxy":
		projectType = app.GoProxyTemplates
	case "skeleton":
		projectType = app.SkeletonTemplates
	}

	var templates template.Templates
	for templateFS, baseDir := range projectType {
		module, err := template.LoadTemplates(templateFS, baseDir)
		if err != nil {
			panic(err)
		}
		templates = append(templates, module...)
	}

	p := generator.NewProject(options.Location, options.Name, app.NewApp(templates))
	err := p.Render()
	if err != nil {
		panic(err)
	}
}
