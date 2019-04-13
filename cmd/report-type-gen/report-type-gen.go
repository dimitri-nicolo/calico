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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"

	yaml "github.com/projectcalico/go-yaml-wrapper"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

func main() {
	const manifestsDir = "manifests"

	// Always start with local "default" directory, unless specified.
	dirs := []string{"default"}
	if len(os.Args) >= 2 {
		dirs = os.Args[1:]
	}

	// Get list of yaml files inside the 1st level of given directories.
	var files []string
	for _, dir := range dirs {
		yamls, err := getYamlFiles(dir)
		if err != nil {
			log.Fatal(err)
		}
		files = append(files, yamls...)
	}

	for _, file := range files {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Error parsing yaml: %v", r)
			}
		}()

		contents, err := ioutil.ReadFile(file)
		if err != nil {
			log.Fatal(err)
		}

		reportType := api.GlobalReportType{}
		if err := yaml.UnmarshalStrict(contents, &reportType); err != nil {
			log.Fatal(err)
		}

		// get the directory for template files.
		inDirName := path.Join(path.Dir(file), reportType.Name)

		if templ, err := getTemplate(inDirName, reportType.Spec.UISummaryTemplate.Name); err == nil {
			reportType.Spec.UISummaryTemplate.Template = string(templ)
		}

		for i := 0; i < len(reportType.Spec.DownloadTemplates); i++ {
			if templ, err := getTemplate(inDirName, reportType.Spec.DownloadTemplates[i].Name); err == nil {
				reportType.Spec.DownloadTemplates[i].Template = string(templ)
			}
		}

		// Generate manifest.
		manifestContent, err := yaml.Marshal(reportType)
		if err != nil {
			log.Fatal(err)
		}
		manifestFullPath := path.Join(path.Dir(file), manifestsDir, path.Base(file))
		if err := ioutil.WriteFile(manifestFullPath, manifestContent, 0644); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Generated ", manifestFullPath)
	}
}

/*
Given a list of directory, return yaml files inside the directory.
*/
func getYamlFiles(dir string) (yamls []string, err error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return yamls, err
	}

	var ret []string
	for _, file := range files {
		// Avoid directories.
		if file.IsDir() {
			continue
		}

		fileName := file.Name()
		fileNameExt := path.Ext(fileName)
		// Process only yaml files.
		if strings.ToLower(fileNameExt) == "yaml" {
			continue
		}

		ret = append(ret, path.Join(dir, fileName))
	}

	return ret, nil
}

/*
Given directory and template-file name, get the contents of the template file.
*/
func getTemplate(dirName string, templName string) (template []byte, err error) {
	templFullPath := path.Join(dirName, templName)
	template, err = ioutil.ReadFile(templFullPath)
	if err != nil {
		return template, err
	}

	return template, nil
}
