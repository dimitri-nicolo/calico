// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package jira

import (
	"bytes"
	"html/template"

	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

type jiraPayload struct {
	Fields jiraFields `json:"fields"`
}

type jiraFields struct {
	Project     jiraProject   `json:"project"`
	IssueType   jiraIssueType `json:"issuetype"`
	Summary     string        `json:"summary"`
	Description string        `json:"description"`
}

type jiraProject struct {
	Key string `json:"key"`
}

type jiraIssueType struct {
	Name string `json:"name"`
}

// TODO: fix .Time and .Record
var descriptionTemplate = template.Must(template.New("description").Parse(`
*Alert type:* {{.Type}}
*Time:* {{.Time}}
*Origin:* {{.Origin}}
*Severity:* {{.Severity}}

{{.Description}}

Detailed information:

{{ range $info, $value := .Record }}
*{{$info}}:* {{$value}}
{{ end }}
`))

func buildSummary(event *lsApi.Event) (string, error) {
	return "Calico Security Alert", nil
}

func buildDescription(event *lsApi.Event) (string, error) {
	buffer := new(bytes.Buffer)
	err := descriptionTemplate.Execute(buffer, event)
	return buffer.String(), err
}
