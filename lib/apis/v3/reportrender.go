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

package v3

import (
	"bytes"
	"text/template"
)

/*
Returns rendered text for given text-template and data struct input.
*/
func RenderTemplate(reportTemplateText string, reportData ReportData) (string, error) {
	var rendered string

	templ, err := template.New("report-template").Parse(reportTemplateText)
	if err != nil {
		return rendered, err
	}

	var b bytes.Buffer
	err = templ.Execute(&b, reportData)
	if err != nil {
		return rendered, err
	}
	rendered = b.String()

	return rendered, nil
}
