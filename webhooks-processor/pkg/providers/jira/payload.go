// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package jira

import (
	"bytes"
	"encoding/json"
	"html/template"
	"strings"
	"time"

	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/sirupsen/logrus"
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

var descriptionTemplate = template.Must(template.New("description").Funcs(template.FuncMap{
	"when": func(when lsApi.TimestampOrDate) string { return when.GetTime().Format(time.RFC850) },
	"record": func(event *lsApi.Event) string {
		data := make(map[string]any)
		if err := event.GetRecord(&data); err != nil {
			logrus.WithError(err).Error("error processing event record")
			return "n/a"
		}
		if bytes, err := json.MarshalIndent(data, "", "\t"); err != nil {
			logrus.WithError(err).Error("error marshalling record data")
			return "n/a"
		} else {
			return strings.ReplaceAll(string(bytes), `"`, `‟`)
		}
	},
}).Parse(`
*What happened:* {{.Description}}
*When it happened:* {{when .Time}}
*Event source:* {{.Origin}}
*Attack vector:* {{.AttackVector}}
*Severity:* {{.Severity}}/100
*Mitre IDs:* {{range .MitreIDs}}{{.}} {{end}}
*Mitre tactic:* {{.MitreTactic}}

*Mitigations:*
{{range .Mitigations}}
- {{.}}{{end}}

{code:json|title=Detailed record information}{{ record .}}{code}
`))

func buildSummary(event *lsApi.Event) (string, error) {
	return "Calico Security Alert", nil
}

func buildDescription(event *lsApi.Event) (string, error) {
	buffer := new(bytes.Buffer)
	err := descriptionTemplate.Execute(buffer, event)
	return buffer.String(), err
}
