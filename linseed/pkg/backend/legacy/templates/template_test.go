// Copyright (c) 2023 Tigera, Inc. All rights reserved.
//

package templates

import (
	"testing"

	"github.com/stretchr/testify/require"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
)

var settings = `{
"lifecycle": {
"name": "tigera_secure_ee_flows_policy",
"rollover_alias": "tigera_secure_ee_flows.test."
}
}`

func TestBoostrapTemplate(t *testing.T) {
	expectedTemplate := &Template{
		IndexPatterns: []string{"tigera_secure_ee_flows*"},
		Mappings:      testutils.MustUnmarshalToMap(t, FlowLogMappings),
		Settings:      testutils.MustUnmarshalToMap(t, settings),
	}

	expectedTemplate.Settings["number_of_shards"] = 1
	expectedTemplate.Settings["number_of_replicas"] = 0

	config := NewTemplateConfig(bapi.FlowLogs, bapi.ClusterInfo{Cluster: "test"}, WithApplication("test-app"))
	template, err := config.Build()
	require.NoError(t, err)
	assertTemplate(t, expectedTemplate, template)
}

func assertTemplate(t *testing.T, expected *Template, template *Template) {
	require.EqualValues(t, expected.IndexPatterns, template.IndexPatterns)
	require.NotEmpty(t, template.Mappings)
	require.EqualValues(t, expected.Mappings, template.Mappings)
	require.NotEmpty(t, template.Settings)
	require.EqualValues(t, expected.Settings["lifecycle"], template.Settings["lifecycle"])
	require.EqualValues(t, expected.Settings["number_shards"], template.Settings["number_shards"])
	require.EqualValues(t, expected.Settings["number_replicas"], template.Settings["number_replicase"])
}
