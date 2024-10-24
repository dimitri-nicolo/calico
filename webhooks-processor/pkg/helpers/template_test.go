// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package helpers_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"

	lsApi "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/webhooks-processor/pkg/helpers"
	"github.com/projectcalico/calico/webhooks-processor/pkg/providers/generic"
)

func TestParseTemplate(t *testing.T) {
	// empty corner case:
	templateData := ""
	_, err := helpers.ParseTemplate(templateData)
	assert.NoError(t, err)

	// valid arbitrary data:
	templateData = `{
	"name": "{{.name}}",
	"event": {{.}}
}`
	_, err = helpers.ParseTemplate(templateData)
	assert.NoError(t, err)

	// invalid template (no data)
	templateData = `{
	"name": "{{.name}}",
	"event": {{}}
}`
	_, err = helpers.ParseTemplate(templateData)
	assert.Error(t, err)

	// invalid template (missing })
	templateData = `{
		"name": "{{.name}}",
		"event": {{}
	}`
	_, err = helpers.ParseTemplate(templateData)
	assert.Error(t, err)

	// invalid template (extra {)
	templateData = `{
		"name": "{{.name}}",
		"event": {{{}}
	}`
	_, err = helpers.ParseTemplate(templateData)
	assert.Error(t, err)
}

func TestProcessTemplate(t *testing.T) {
	event := lsApi.Event{
		ID:          "testid",
		Description: "This is an event",
		Severity:    23,
		Time:        lsApi.NewEventTimestamp(time.Now().Unix()),
		Type:        "runtime_security",
	}
	labels := map[string]string{
		"cluster": "test-cluster",
	}
	payload, err := json.Marshal(generic.GenericProviderPayload{Event: &event, Labels: labels})
	assert.NoError(t, err)

	templateData := `{
	"message": "We're testing payload templating",
	"event": "{{.description}}",
	"event_type": "{{.type}}",
	"cluster-id": "{{.labels.cluster}}",
	"missing_data": "{{.this_field_does_not_exists}}",
	"nested_missing_data": "{{.this_field_does_not_exists.nested}}",
	"don't_do_this": "{{.}}"
}`
	tmpl, err := helpers.ParseTemplate(templateData)
	assert.NoError(t, err)
	assert.NotNil(t, tmpl)

	result, err := helpers.ProcessTemplate(tmpl, payload)
	assert.NoError(t, err)

	resultJson := string(result)
	assert.Equal(t, event.Description, gjson.Get(resultJson, "event").String())
	assert.Equal(t, event.Type, gjson.Get(resultJson, "event_type").String())
	assert.Equal(t, labels["cluster"], gjson.Get(resultJson, "cluster-id").String())
	assert.Equal(t, "", gjson.Get(resultJson, "missing_data").String())
	assert.Equal(t, "", gjson.Get(resultJson, "nested_missing_data").String())

	// This is a string representation of the original map, NOT the original JSON
	// We should not document/encourage this as it's not what we might expect
	assert.Contains(t, gjson.Get(resultJson, "don't_do_this").String(), "map[description:This is an event geo_info:map[] id:testid labels:map[cluster:test-cluster] ")
}
