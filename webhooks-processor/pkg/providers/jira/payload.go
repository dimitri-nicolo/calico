// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package jira

import (
	"bytes"
	"encoding/json"
	"html/template"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

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

var descriptionTemplate = template.Must(template.New("description").Funcs(template.FuncMap{
	"when": func(when lsApi.TimestampOrDate) string { return when.GetTime().Format(time.RFC850) },
	"join": func(pieces []string) string { return strings.Join(pieces, " ") },
	"record": func(event *lsApi.Event) string {
		data := make(map[string]any)
		if err := event.GetRecord(&data); err != nil {
			logrus.WithError(err).Error("error processing event record")
			return "n/a"
		} else if data == nil {
			return "n/a"
		} else if bytes, err := json.MarshalIndent(data, "", "\t"); err != nil {
			logrus.WithError(err).Error("error marshalling record data")
			return "n/a"
		} else {
			// The following characters of the record will get encoded down the line:
			// " < > [ \ ] ^ ` { | }
			// The reason is the structure of what we are sending to the Jira endpoint:
			// - a JSON payload
			// - that contains some formatted text
			// - with another JSON (record data) embedded in it.
			// This could be rectified on Jira end but we don't have any control over it.
			// There is no workaround to this issue and the best we can do is to convert these
			// characters to their UTF-8 equivalents to ensure the document is properly displayed.
			// The trade-off is that the displayed record will no longer be a valid JSON document.
			record := string(bytes)
			record = strings.ReplaceAll(record, `"`, "ʺ")
			record = strings.ReplaceAll(record, "<", "ᐸ")
			record = strings.ReplaceAll(record, ">", "ᐳ")
			record = strings.ReplaceAll(record, "[", "❲")
			record = strings.ReplaceAll(record, `\`, "∖")
			record = strings.ReplaceAll(record, "]", "❳")
			record = strings.ReplaceAll(record, "^", "˄")
			record = strings.ReplaceAll(record, "`", "ʽ")
			record = strings.ReplaceAll(record, "{", "❴")
			record = strings.ReplaceAll(record, "|", "ǀ")
			record = strings.ReplaceAll(record, "}", "❵")
			return record
		}
	},
}).Parse(`
*What happened:* {{.Description}}
*When it happened:* {{when .Time}}
*Event source:* {{.Origin}}
*Attack vector:* {{.AttackVector}}
*Severity:* {{.Severity}}/100
*Mitre IDs:* {{join .MitreIDs}}
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
