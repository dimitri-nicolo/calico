// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package templates

import (
	"fmt"
	"testing"

	backendutils "github.com/projectcalico/calico/linseed/pkg/backend/testutils"

	"github.com/projectcalico/calico/linseed/pkg/testutils"

	"github.com/stretchr/testify/require"

	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
)

func TestBoostrapFlowsTemplate(t *testing.T) {
	cluster := backendutils.RandomClusterName()

	settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_flows_policy",
"rollover_alias": "tigera_secure_ee_flows.%s."
}
}`, cluster)

	expectedTemplate := &Template{
		IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_flows.%s.*", cluster)},
		Mappings:      testutils.MustUnmarshalToMap(t, FlowLogMappings),
		Settings:      testutils.MustUnmarshalToMap(t, settings),
	}

	expectedTemplate.Settings["number_of_shards"] = 1
	expectedTemplate.Settings["number_of_replicas"] = 0

	config := NewTemplateConfig(bapi.FlowLogs, bapi.ClusterInfo{Cluster: cluster}, WithApplication("test-app"))
	require.Equal(t, fmt.Sprintf("tigera_secure_ee_flows.%s.", cluster), config.TemplateName())
	template, err := config.Build()
	require.NoError(t, err)
	assertTemplate(t, expectedTemplate, template)
}

func TestBoostrapFlowsMultiTenantTemplate(t *testing.T) {
	cluster := backendutils.RandomClusterName()
	tenant := backendutils.RandomTenantName()

	settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_flows_policy",
"rollover_alias": "tigera_secure_ee_flows.%s.%s."
}
}`, tenant, cluster)

	expectedTemplate := &Template{
		IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_flows.%s.%s.*", tenant, cluster)},
		Mappings:      testutils.MustUnmarshalToMap(t, FlowLogMappings),
		Settings:      testutils.MustUnmarshalToMap(t, settings),
	}

	expectedTemplate.Settings["number_of_shards"] = 1
	expectedTemplate.Settings["number_of_replicas"] = 0

	config := NewTemplateConfig(bapi.FlowLogs, bapi.ClusterInfo{Cluster: cluster, Tenant: tenant}, WithApplication("test-app"))
	require.Equal(t, fmt.Sprintf("tigera_secure_ee_flows.%s.%s.", tenant, cluster), config.TemplateName())
	template, err := config.Build()
	require.NoError(t, err)
	assertTemplate(t, expectedTemplate, template)
}

func TestBoostrapDNSTemplate(t *testing.T) {
	cluster := backendutils.RandomClusterName()

	expectedTemplate := &Template{
		IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_dns.%s.*", cluster)},
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
	require.Equal(t, fmt.Sprintf("tigera_secure_ee_dns.%s.", cluster), config.TemplateName())
	template, err := config.Build()
	require.NoError(t, err)
	assertTemplate(t, expectedTemplate, template)
}

func TestBoostrapEEAuditTemplate(t *testing.T) {
	cluster := backendutils.RandomClusterName()

	settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_audit_ee_policy",
"rollover_alias": "tigera_secure_ee_audit_ee.%s."
}
}`, cluster)

	expectedTemplate := &Template{
		IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_audit_ee.%s.*", cluster)},
		Mappings:      testutils.MustUnmarshalToMap(t, AuditMappings),
		Settings:      testutils.MustUnmarshalToMap(t, settings),
	}

	expectedTemplate.Settings["number_of_shards"] = 1
	expectedTemplate.Settings["number_of_replicas"] = 0

	config := NewTemplateConfig(bapi.AuditEELogs, bapi.ClusterInfo{Cluster: cluster}, WithApplication("test-app"))
	require.Equal(t, fmt.Sprintf("tigera_secure_ee_audit_ee.%s.", cluster), config.TemplateName())
	template, err := config.Build()
	require.NoError(t, err)
	assertTemplate(t, expectedTemplate, template)
}

func TestBoostrapKUBEAuditTemplate(t *testing.T) {
	cluster := backendutils.RandomClusterName()

	settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_audit_kube_policy",
"rollover_alias": "tigera_secure_ee_audit_kube.%s."
}
}`, cluster)

	expectedTemplate := &Template{
		IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_audit_kube.%s.*", cluster)},
		Mappings:      testutils.MustUnmarshalToMap(t, AuditMappings),
		Settings:      testutils.MustUnmarshalToMap(t, settings),
	}

	expectedTemplate.Settings["number_of_shards"] = 1
	expectedTemplate.Settings["number_of_replicas"] = 0

	config := NewTemplateConfig(bapi.AuditKubeLogs, bapi.ClusterInfo{Cluster: cluster}, WithApplication("test-app"))
	require.Equal(t, fmt.Sprintf("tigera_secure_ee_audit_kube.%s.", cluster), config.TemplateName())
	template, err := config.Build()
	require.NoError(t, err)
	assertTemplate(t, expectedTemplate, template)
}

func TestBoostrapBGPTemplate(t *testing.T) {
	cluster := backendutils.RandomClusterName()

	settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_bgp_policy",
"rollover_alias": "tigera_secure_ee_bgp.%s."
}
}`, cluster)

	expectedTemplate := &Template{
		IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_bgp.%s.*", cluster)},
		Mappings:      testutils.MustUnmarshalToMap(t, BGPMappings),
		Settings:      testutils.MustUnmarshalToMap(t, settings),
	}

	expectedTemplate.Settings["number_of_shards"] = 1
	expectedTemplate.Settings["number_of_replicas"] = 0

	config := NewTemplateConfig(bapi.BGPLogs, bapi.ClusterInfo{Cluster: cluster}, WithApplication("test-app"))
	require.Equal(t, fmt.Sprintf("tigera_secure_ee_bgp.%s.", cluster), config.TemplateName())
	template, err := config.Build()
	require.NoError(t, err)
	assertTemplate(t, expectedTemplate, template)
}

func TestBoostrapL7Template(t *testing.T) {
	cluster := backendutils.RandomClusterName()

	settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_l7_policy",
"rollover_alias": "tigera_secure_ee_l7.%s."
}
}`, cluster)

	expectedTemplate := &Template{
		IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_l7.%s.*", cluster)},
		Mappings:      testutils.MustUnmarshalToMap(t, L7LogMappings),
		Settings:      testutils.MustUnmarshalToMap(t, settings),
	}

	expectedTemplate.Settings["number_of_shards"] = 1
	expectedTemplate.Settings["number_of_replicas"] = 0

	config := NewTemplateConfig(bapi.L7Logs, bapi.ClusterInfo{Cluster: cluster}, WithApplication("test-app"))
	require.Equal(t, fmt.Sprintf("tigera_secure_ee_l7.%s.", cluster), config.TemplateName())
	template, err := config.Build()
	require.NoError(t, err)
	assertTemplate(t, expectedTemplate, template)
}

func TestBoostrapWAFTemplate(t *testing.T) {
	cluster := backendutils.RandomClusterName()

	settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_waf_policy",
"rollover_alias": "tigera_secure_ee_waf.%s."
}
}`, cluster)

	expectedTemplate := &Template{
		IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_waf.%s.*", cluster)},
		Mappings:      testutils.MustUnmarshalToMap(t, WAFMappings),
		Settings:      testutils.MustUnmarshalToMap(t, settings),
	}

	expectedTemplate.Settings["number_of_shards"] = 1
	expectedTemplate.Settings["number_of_replicas"] = 0

	config := NewTemplateConfig(bapi.WAFLogs, bapi.ClusterInfo{Cluster: cluster}, WithApplication("test-app"))
	require.Equal(t, fmt.Sprintf("tigera_secure_ee_waf.%s.", cluster), config.TemplateName())
	template, err := config.Build()
	require.NoError(t, err)
	assertTemplate(t, expectedTemplate, template)
}

func TestBoostrapRuntimeReportsTemplate(t *testing.T) {
	cluster := backendutils.RandomClusterName()

	settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_runtime_policy",
"rollover_alias": "tigera_secure_ee_runtime.%s."
}
}`, cluster)

	expectedTemplate := &Template{
		IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_runtime.%s.*", cluster)},
		Mappings:      testutils.MustUnmarshalToMap(t, RuntimeReportsMappings),
		Settings:      testutils.MustUnmarshalToMap(t, settings),
	}

	expectedTemplate.Settings["number_of_shards"] = 1
	expectedTemplate.Settings["number_of_replicas"] = 0

	config := NewTemplateConfig(bapi.RuntimeReports, bapi.ClusterInfo{Cluster: cluster}, WithApplication("test-app"))
	require.Equal(t, fmt.Sprintf("tigera_secure_ee_runtime.%s.", cluster), config.TemplateName())
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
