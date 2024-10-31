// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package main

import (
	"flag"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/compliance-configuration-security/control-details-scraper/pkg/controlgen"
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
