// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package main

import (
	"flag"

	"github.com/projectcalico/calico/ccs/control-details-scraper/pkg/controlgen"
	"github.com/sirupsen/logrus"
)

func main() {
	regoLibPath := flag.String("regoPath", "./regolibrary", "repository to read control definitions from")
	writeToPath := flag.String("writeToPath", "", "writes ccd control details here")
	flag.Parse()

	cdGenerator := controlgen.NewControlsGenerator(*regoLibPath, *writeToPath)

	err := cdGenerator.Generate()
	if err != nil {
		logrus.Fatal(err)
	}

}
