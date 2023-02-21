// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package templates

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
)

func TestBoostrapFlowsTemplate(t *testing.T) {
	cluster := testutils.RandomClusterName()

	settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_flows_policy",
"rollover_alias": "tigera_secure_ee_flows.%s."
}
}`, cluster)

	expectedTemplate := &Template{
		IndexPatterns: []string{"tigera_secure_ee_flows*"},
		Mappings:      testutils.MustUnmarshalToMap(t, FlowLogMappings),
		Settings:      testutils.MustUnmarshalToMap(t, settings),
	}

	expectedTemplate.Settings["number_of_shards"] = 1
	expectedTemplate.Settings["number_of_replicas"] = 0

	config := NewTemplateConfig(bapi.FlowLogs, bapi.ClusterInfo{Cluster: cluster}, WithApplication("test-app"))
	template, err := config.Build()
	require.NoError(t, err)
	assertTemplate(t, expectedTemplate, template)
}

func TestBoostrapDNSTemplate(t *testing.T) {
	cluster := testutils.RandomClusterName()

	expectedTemplate := &Template{
		IndexPatterns: []string{"tigera_secure_ee_dns*"},
		Mappings:      testutils.MustUnmarshalToMap(t, DNSLogMappings),
		Settings:      map[string]interface{}{},
	}

	// DNS has additional settings that we need to take into account
	expectedTemplate.Settings = testutils.MustUnmarshalToMap(t, DNSLogSettings)
	expectedTemplate.Settings["lifecycle"] = map[string]interface{}{
		"name":           "tigera_secure_ee_dns_policy",
		"rollover_alias": fmt.Sprintf("tigera_secure_ee_dns.%s.", cluster),
	}
	expectedTemplate.Settings["number_of_shards"] = 1
	expectedTemplate.Settings["number_of_replicas"] = 0

	config := NewTemplateConfig(bapi.DNSLogs, bapi.ClusterInfo{Cluster: cluster}, WithApplication("test-app"))
	template, err := config.Build()
	require.NoError(t, err)
	assertTemplate(t, expectedTemplate, template)
}

func TestBoostrapEEAuditTemplate(t *testing.T) {
	cluster := testutils.RandomClusterName()

	settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_audit_ee_policy",
"rollover_alias": "tigera_secure_ee_audit_ee.%s."
}
}`, cluster)

	expectedTemplate := &Template{
		IndexPatterns: []string{"tigera_secure_ee_audit_*"},
		Mappings:      testutils.MustUnmarshalToMap(t, AuditMappings),
		Settings:      testutils.MustUnmarshalToMap(t, settings),
	}

	expectedTemplate.Settings["number_of_shards"] = 1
	expectedTemplate.Settings["number_of_replicas"] = 0

	config := NewTemplateConfig(bapi.AuditEELogs, bapi.ClusterInfo{Cluster: cluster}, WithApplication("test-app"))
	template, err := config.Build()
	require.NoError(t, err)
	assertTemplate(t, expectedTemplate, template)
}

func TestBoostrapKUBEAuditTemplate(t *testing.T) {
	cluster := testutils.RandomClusterName()

	settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_audit_kube_policy",
"rollover_alias": "tigera_secure_ee_audit_kube.%s."
}
}`, cluster)

	expectedTemplate := &Template{
		IndexPatterns: []string{"tigera_secure_ee_audit_*"},
		Mappings:      testutils.MustUnmarshalToMap(t, AuditMappings),
		Settings:      testutils.MustUnmarshalToMap(t, settings),
	}

	expectedTemplate.Settings["number_of_shards"] = 1
	expectedTemplate.Settings["number_of_replicas"] = 0

	config := NewTemplateConfig(bapi.AuditKubeLogs, bapi.ClusterInfo{Cluster: cluster}, WithApplication("test-app"))
	template, err := config.Build()
	require.NoError(t, err)
	assertTemplate(t, expectedTemplate, template)
}

func assertTemplate(t *testing.T, expected *Template, template *Template) {
	require.EqualValues(t, expected.IndexPatterns, template.IndexPatterns)
	require.NotEmpty(t, template.Mappings)
	require.EqualValues(t, expected.Mappings, template.Mappings)
	require.NotEmpty(t, template.Settings)
	require.EqualValues(t, expected.Settings, template.Settings)
}
