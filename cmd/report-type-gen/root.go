// Copyright (c) 2019 Tigera, Inc. All rights reserved.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "report-type-gen",
		Short: "Maintains compliance report-type manifests",
		Long: `
A simple utilty that reads basic report-type structure and components in the specified directory to generate directly usable manifests.
Generated manifests are stored in the 'manifest/' directory in the current location.`,
	}

	inDir   string
	outDir  string
	viewDir string
)

func init() {
	rootCmd.AddCommand(genCmd)
	genCmd.Flags().StringVarP(&inDir, "input", "i", "./report-types/default", "input directory containing basic report-type structure.")
	genCmd.Flags().StringVarP(&outDir, "output", "o", "./output/default", "output directory containing generated manifests and code snippets.")

	rootCmd.AddCommand(viewCmd)
	viewCmd.Flags().StringVarP(&viewDir, "input", "i", "./manifests/default", "input directory containing usable report-type manifests.")
}

func Execute() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.ErrorLevel)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
