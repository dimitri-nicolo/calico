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
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("TestCreateFlowsTemplate (tenant=%s)", tenant)
		t.Run(name, func(t *testing.T) {
			cluster := backendutils.RandomClusterName()

			settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_flows_policy",
"rollover_alias": "tigera_secure_ee_flows.%s."
}
}`, print(tenant, cluster))

			expectedTemplate := &Template{
				IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_flows.%s.*", print(tenant, cluster))},
				Mappings:      testutils.MustUnmarshalToMap(t, FlowLogMappings),
				Settings:      testutils.MustUnmarshalToMap(t, settings),
			}

			expectedTemplate.Settings["number_of_shards"] = 1
			expectedTemplate.Settings["number_of_replicas"] = 0

			config := NewTemplateConfig(bapi.FlowLogs, bapi.ClusterInfo{Cluster: cluster, Tenant: tenant})
			require.Equal(t, fmt.Sprintf("tigera_secure_ee_flows.%s.", print(tenant, cluster)), config.TemplateName())
			template, err := config.Template()
			require.NoError(t, err)
			assertTemplate(t, expectedTemplate, template)

			require.Equal(t, fmt.Sprintf("tigera_secure_ee_flows.%s.", print(tenant, cluster)), config.Alias())
			require.Equal(t, fmt.Sprintf("<tigera_secure_ee_flows.%s.fluentd-{now/s{yyyyMMdd}}-000001>",
				print(tenant, cluster)), config.BootstrapIndexName())

		})
	}
}

func print(tenant, cluster string) string {
	if tenant == "" {
		return cluster
	}

	return fmt.Sprintf("%s.%s", tenant, cluster)
}

func TestBoostrapDNSTemplate(t *testing.T) {
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("TestCreateDNSTemplate (tenant=%s)", tenant)
		t.Run(name, func(t *testing.T) {
			cluster := backendutils.RandomClusterName()

			expectedTemplate := &Template{
				IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_dns.%s.*", print(tenant, cluster))},
				Mappings:      testutils.MustUnmarshalToMap(t, DNSLogMappings),
				Settings:      map[string]interface{}{},
			}

			// DNS has additional settings that we need to take into account
			expectedTemplate.Settings = testutils.MustUnmarshalToMap(t, DNSLogSettings)
			expectedTemplate.Settings["lifecycle"] = map[string]interface{}{
				"name":           "tigera_secure_ee_dns_policy",
				"rollover_alias": fmt.Sprintf("tigera_secure_ee_dns.%s.", print(tenant, cluster)),
			}
			expectedTemplate.Settings["number_of_shards"] = 1
			expectedTemplate.Settings["number_of_replicas"] = 0

			config := NewTemplateConfig(bapi.DNSLogs, bapi.ClusterInfo{Cluster: cluster, Tenant: tenant})
			require.Equal(t, fmt.Sprintf("tigera_secure_ee_dns.%s.", print(tenant, cluster)), config.TemplateName())
			template, err := config.Template()
			require.NoError(t, err)
			assertTemplate(t, expectedTemplate, template)

			require.Equal(t, fmt.Sprintf("tigera_secure_ee_dns.%s.", print(tenant, cluster)), config.Alias())
			require.Equal(t, fmt.Sprintf("<tigera_secure_ee_dns.%s.fluentd-{now/s{yyyyMMdd}}-000001>",
				print(tenant, cluster)), config.BootstrapIndexName())
		})
	}
}

func TestBoostrapEEAuditTemplate(t *testing.T) {
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("TestCreateEEAuditTemplate (tenant=%s)", tenant)
		t.Run(name, func(t *testing.T) {

			cluster := backendutils.RandomClusterName()

			settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_audit_ee_policy",
"rollover_alias": "tigera_secure_ee_audit_ee.%s."
}
}`, print(tenant, cluster))

			expectedTemplate := &Template{
				IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_audit_ee.%s.*", print(tenant, cluster))},
				Mappings:      testutils.MustUnmarshalToMap(t, AuditMappings),
				Settings:      testutils.MustUnmarshalToMap(t, settings),
			}

			expectedTemplate.Settings["number_of_shards"] = 1
			expectedTemplate.Settings["number_of_replicas"] = 0

			config := NewTemplateConfig(bapi.AuditEELogs, bapi.ClusterInfo{Cluster: cluster, Tenant: tenant})
			require.Equal(t, fmt.Sprintf("tigera_secure_ee_audit_ee.%s.", print(tenant, cluster)), config.TemplateName())
			template, err := config.Template()
			require.NoError(t, err)
			assertTemplate(t, expectedTemplate, template)

			require.Equal(t, fmt.Sprintf("tigera_secure_ee_audit_ee.%s.", print(tenant, cluster)), config.Alias())
			require.Equal(t, fmt.Sprintf("<tigera_secure_ee_audit_ee.%s.fluentd-{now/s{yyyyMMdd}}-000001>",
				print(tenant, cluster)), config.BootstrapIndexName())

		})
	}
}

func TestBoostrapKUBEAuditTemplate(t *testing.T) {
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("TestCreateKubeAuditTemplate (tenant=%s)", tenant)
		t.Run(name, func(t *testing.T) {

			cluster := backendutils.RandomClusterName()

			settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_audit_kube_policy",
"rollover_alias": "tigera_secure_ee_audit_kube.%s."
}
}`, print(tenant, cluster))

			expectedTemplate := &Template{
				IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_audit_kube.%s.*", print(tenant, cluster))},
				Mappings:      testutils.MustUnmarshalToMap(t, AuditMappings),
				Settings:      testutils.MustUnmarshalToMap(t, settings),
			}

			expectedTemplate.Settings["number_of_shards"] = 1
			expectedTemplate.Settings["number_of_replicas"] = 0

			config := NewTemplateConfig(bapi.AuditKubeLogs, bapi.ClusterInfo{Cluster: cluster, Tenant: tenant})
			require.Equal(t, fmt.Sprintf("tigera_secure_ee_audit_kube.%s.", print(tenant, cluster)), config.TemplateName())
			template, err := config.Template()
			require.NoError(t, err)
			assertTemplate(t, expectedTemplate, template)

			require.Equal(t, fmt.Sprintf("tigera_secure_ee_audit_kube.%s.", print(tenant, cluster)), config.Alias())
			require.Equal(t, fmt.Sprintf("<tigera_secure_ee_audit_kube.%s.fluentd-{now/s{yyyyMMdd}}-000001>",
				print(tenant, cluster)), config.BootstrapIndexName())

		})
	}
}

func TestBoostrapBGPTemplate(t *testing.T) {
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("TestCreateBGPTemplate (tenant=%s)", tenant)
		t.Run(name, func(t *testing.T) {

			cluster := backendutils.RandomClusterName()

			settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_bgp_policy",
"rollover_alias": "tigera_secure_ee_bgp.%s."
}
}`, print(tenant, cluster))

			expectedTemplate := &Template{
				IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_bgp.%s.*", print(tenant, cluster))},
				Mappings:      testutils.MustUnmarshalToMap(t, BGPMappings),
				Settings:      testutils.MustUnmarshalToMap(t, settings),
			}

			expectedTemplate.Settings["number_of_shards"] = 1
			expectedTemplate.Settings["number_of_replicas"] = 0

			config := NewTemplateConfig(bapi.BGPLogs, bapi.ClusterInfo{Cluster: cluster, Tenant: tenant})
			require.Equal(t, fmt.Sprintf("tigera_secure_ee_bgp.%s.", print(tenant, cluster)), config.TemplateName())
			template, err := config.Template()
			require.NoError(t, err)
			assertTemplate(t, expectedTemplate, template)

			require.Equal(t, fmt.Sprintf("tigera_secure_ee_bgp.%s.", print(tenant, cluster)), config.Alias())
			require.Equal(t, fmt.Sprintf("<tigera_secure_ee_bgp.%s.fluentd-{now/s{yyyyMMdd}}-000001>",
				print(tenant, cluster)), config.BootstrapIndexName())

		})
	}

}

func TestBoostrapL7Template(t *testing.T) {
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("TestCreateL7Template (tenant=%s)", tenant)
		t.Run(name, func(t *testing.T) {

			cluster := backendutils.RandomClusterName()

			settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_l7_policy",
"rollover_alias": "tigera_secure_ee_l7.%s."
}
}`, print(tenant, cluster))

			expectedTemplate := &Template{
				IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_l7.%s.*", print(tenant, cluster))},
				Mappings:      testutils.MustUnmarshalToMap(t, L7LogMappings),
				Settings:      testutils.MustUnmarshalToMap(t, settings),
			}

			expectedTemplate.Settings["number_of_shards"] = 1
			expectedTemplate.Settings["number_of_replicas"] = 0

			config := NewTemplateConfig(bapi.L7Logs, bapi.ClusterInfo{Cluster: cluster, Tenant: tenant})
			require.Equal(t, fmt.Sprintf("tigera_secure_ee_l7.%s.", print(tenant, cluster)), config.TemplateName())
			template, err := config.Template()
			require.NoError(t, err)
			assertTemplate(t, expectedTemplate, template)

			require.Equal(t, fmt.Sprintf("tigera_secure_ee_l7.%s.", print(tenant, cluster)), config.Alias())
			require.Equal(t, fmt.Sprintf("<tigera_secure_ee_l7.%s.fluentd-{now/s{yyyyMMdd}}-000001>",
				print(tenant, cluster)), config.BootstrapIndexName())

		})
	}
}

func TestBoostrapWAFTemplate(t *testing.T) {
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("TestCreateWAFTemplate (tenant=%s)", tenant)
		t.Run(name, func(t *testing.T) {

			cluster := backendutils.RandomClusterName()

			settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_waf_policy",
"rollover_alias": "tigera_secure_ee_waf.%s."
}
}`, print(tenant, cluster))

			expectedTemplate := &Template{
				IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_waf.%s.*", print(tenant, cluster))},
				Mappings:      testutils.MustUnmarshalToMap(t, WAFMappings),
				Settings:      testutils.MustUnmarshalToMap(t, settings),
			}

			expectedTemplate.Settings["number_of_shards"] = 1
			expectedTemplate.Settings["number_of_replicas"] = 0

			config := NewTemplateConfig(bapi.WAFLogs, bapi.ClusterInfo{Cluster: cluster, Tenant: tenant})
			require.Equal(t, fmt.Sprintf("tigera_secure_ee_waf.%s.", print(tenant, cluster)), config.TemplateName())
			template, err := config.Template()
			require.NoError(t, err)
			assertTemplate(t, expectedTemplate, template)

			require.Equal(t, fmt.Sprintf("tigera_secure_ee_waf.%s.", print(tenant, cluster)), config.Alias())
			require.Equal(t, fmt.Sprintf("<tigera_secure_ee_waf.%s.fluentd-{now/s{yyyyMMdd}}-000001>",
				print(tenant, cluster)), config.BootstrapIndexName())

		})
	}
}

func TestBoostrapRuntimeReportsTemplate(t *testing.T) {
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("TestCreateRuntimeReportsTemplate (tenant=%s)", tenant)
		t.Run(name, func(t *testing.T) {

			cluster := backendutils.RandomClusterName()

			settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_runtime_policy",
"rollover_alias": "tigera_secure_ee_runtime.%s."
}
}`, print(tenant, cluster))

			expectedTemplate := &Template{
				IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_runtime.%s.*", print(tenant, cluster))},
				Mappings:      testutils.MustUnmarshalToMap(t, RuntimeReportsMappings),
				Settings:      testutils.MustUnmarshalToMap(t, settings),
			}

			expectedTemplate.Settings["number_of_shards"] = 1
			expectedTemplate.Settings["number_of_replicas"] = 0

			config := NewTemplateConfig(bapi.RuntimeReports, bapi.ClusterInfo{Cluster: cluster, Tenant: tenant})
			require.Equal(t, fmt.Sprintf("tigera_secure_ee_runtime.%s.", print(tenant, cluster)), config.TemplateName())
			template, err := config.Template()
			require.NoError(t, err)
			assertTemplate(t, expectedTemplate, template)

			require.Equal(t, fmt.Sprintf("tigera_secure_ee_runtime.%s.", print(tenant, cluster)), config.Alias())
			require.Equal(t, fmt.Sprintf("<tigera_secure_ee_runtime.%s.fluentd-{now/s{yyyyMMdd}}-000001>",
				print(tenant, cluster)), config.BootstrapIndexName())

		})
	}
}

func TestBoostrapComplianceReportsTemplate(t *testing.T) {
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("TestCreateComplianceReportsTemplate (tenant=%s)", tenant)
		t.Run(name, func(t *testing.T) {

			cluster := backendutils.RandomClusterName()

			settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_compliance_reports_policy",
"rollover_alias": "tigera_secure_ee_compliance_reports.%s."
}
}`, print(tenant, cluster))

			expectedTemplate := &Template{
				IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_compliance_reports.%s.*", print(tenant, cluster))},
				Mappings:      testutils.MustUnmarshalToMap(t, ReportMappings),
				Settings:      testutils.MustUnmarshalToMap(t, settings),
			}

			expectedTemplate.Settings["number_of_shards"] = 1
			expectedTemplate.Settings["number_of_replicas"] = 0

			config := NewTemplateConfig(bapi.ReportData, bapi.ClusterInfo{Cluster: cluster, Tenant: tenant})
			require.Equal(t, fmt.Sprintf("tigera_secure_ee_compliance_reports.%s", print(tenant, cluster)), config.TemplateName())
			template, err := config.Template()
			require.NoError(t, err)
			assertTemplate(t, expectedTemplate, template)

			require.Equal(t, fmt.Sprintf("tigera_secure_ee_compliance_reports.%s.", print(tenant, cluster)), config.Alias())
			require.Equal(t, fmt.Sprintf("<tigera_secure_ee_compliance_reports.%s.lma-{now/s{yyyyMMdd}}-000000>",
				print(tenant, cluster)), config.BootstrapIndexName())

		})
	}
}

func TestBoostrapComplianceBenchmarksTemplate(t *testing.T) {
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("TestCreateComplianceBenchmarksTemplate (tenant=%s)", tenant)
		t.Run(name, func(t *testing.T) {

			cluster := backendutils.RandomClusterName()

			settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_benchmark_results_policy",
"rollover_alias": "tigera_secure_ee_benchmark_results.%s."
}
}`, print(tenant, cluster))

			expectedTemplate := &Template{
				IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_benchmark_results.%s.*", print(tenant, cluster))},
				Mappings:      testutils.MustUnmarshalToMap(t, BenchmarksMappings),
				Settings:      testutils.MustUnmarshalToMap(t, settings),
			}

			expectedTemplate.Settings["number_of_shards"] = 1
			expectedTemplate.Settings["number_of_replicas"] = 0

			config := NewTemplateConfig(bapi.Benchmarks, bapi.ClusterInfo{Cluster: cluster, Tenant: tenant})
			require.Equal(t, fmt.Sprintf("tigera_secure_ee_benchmark_results.%s", print(tenant, cluster)), config.TemplateName())
			template, err := config.Template()
			require.NoError(t, err)
			assertTemplate(t, expectedTemplate, template)

			require.Equal(t, fmt.Sprintf("tigera_secure_ee_benchmark_results.%s.", print(tenant, cluster)), config.Alias())
			require.Equal(t, fmt.Sprintf("<tigera_secure_ee_benchmark_results.%s.lma-{now/s{yyyyMMdd}}-000000>",
				print(tenant, cluster)), config.BootstrapIndexName())

		})
	}
}

func TestBoostrapComplianceSnapshotsTemplate(t *testing.T) {
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("TestCreateComplianceSnapshotsTemplate (tenant=%s)", tenant)
		t.Run(name, func(t *testing.T) {

			cluster := backendutils.RandomClusterName()

			settings := fmt.Sprintf(`{
"lifecycle": {
"name": "tigera_secure_ee_snapshots_policy",
"rollover_alias": "tigera_secure_ee_snapshots.%s."
}
}`, print(tenant, cluster))

			expectedTemplate := &Template{
				IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_snapshots.%s.*", print(tenant, cluster))},
				Mappings:      testutils.MustUnmarshalToMap(t, SnapshotMappings),
				Settings:      testutils.MustUnmarshalToMap(t, settings),
			}

			expectedTemplate.Settings["number_of_shards"] = 1
			expectedTemplate.Settings["number_of_replicas"] = 0

			config := NewTemplateConfig(bapi.Snapshots, bapi.ClusterInfo{Cluster: cluster, Tenant: tenant})
			require.Equal(t, fmt.Sprintf("tigera_secure_ee_snapshots.%s", print(tenant, cluster)), config.TemplateName())
			template, err := config.Template()
			require.NoError(t, err)
			assertTemplate(t, expectedTemplate, template)

			require.Equal(t, fmt.Sprintf("tigera_secure_ee_snapshots.%s.", print(tenant, cluster)), config.Alias())
			require.Equal(t, fmt.Sprintf("<tigera_secure_ee_snapshots.%s.lma-{now/s{yyyyMMdd}}-000000>",
				print(tenant, cluster)), config.BootstrapIndexName())

		})
	}
}

func TestBoostrapEventsTemplate(t *testing.T) {
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("TestCreateEventsTemplate (tenant=%s)", tenant)
		t.Run(name, func(t *testing.T) {

			cluster := backendutils.RandomClusterName()

			settings := fmt.Sprintf(`{
	"analysis":{
		"normalizer":{"lowercase":{"filter":"lowercase", "type":"custom"}}
	}, "lifecycle":{
		"name":"tigera_secure_ee_events_policy",
		"rollover_alias":"tigera_secure_ee_events.%s."
	}}`, print(tenant, cluster))

			expectedTemplate := &Template{
				IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_events.%s.*", print(tenant, cluster))},
				Mappings:      testutils.MustUnmarshalToMap(t, EventsMappings),
				Settings:      testutils.MustUnmarshalToMap(t, settings),
			}

			expectedTemplate.Settings["number_of_shards"] = 1
			expectedTemplate.Settings["number_of_replicas"] = 0

			config := NewTemplateConfig(bapi.Events, bapi.ClusterInfo{Cluster: cluster, Tenant: tenant})
			require.Equal(t, fmt.Sprintf("tigera_secure_ee_events.%s", print(tenant, cluster)), config.TemplateName())
			template, err := config.Template()
			require.NoError(t, err)
			assertTemplate(t, expectedTemplate, template)

			require.Equal(t, fmt.Sprintf("tigera_secure_ee_events.%s.", print(tenant, cluster)), config.Alias())
			require.Equal(t, fmt.Sprintf("<tigera_secure_ee_events.%s.lma-{now/s{yyyyMMdd}}-000000>",
				print(tenant, cluster)), config.BootstrapIndexName())

		})
	}
}

func TestBoostrapIPSetTemplate(t *testing.T) {
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("TestCreateIPSetTemplate (tenant=%s)", tenant)
		t.Run(name, func(t *testing.T) {

			cluster := backendutils.RandomClusterName()

			expectedTemplate := &Template{
				IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_threatfeeds_ipset.%s.*", print(tenant, cluster))},
				Mappings:      testutils.MustUnmarshalToMap(t, IPSetMappings),
				Settings:      map[string]interface{}{},
			}

			expectedTemplate.Settings["number_of_shards"] = 1
			expectedTemplate.Settings["number_of_replicas"] = 0

			config := NewTemplateConfig(bapi.IPSet, bapi.ClusterInfo{Cluster: cluster, Tenant: tenant})
			require.Equal(t, fmt.Sprintf("tigera_secure_ee_threatfeeds_ipset.%s", print(tenant, cluster)), config.TemplateName())
			template, err := config.Template()
			require.NoError(t, err)
			assertTemplate(t, expectedTemplate, template)

			require.Equal(t, fmt.Sprintf("tigera_secure_ee_threatfeeds_ipset.%s.", print(tenant, cluster)), config.Alias())
			require.Equal(t, fmt.Sprintf("<tigera_secure_ee_threatfeeds_ipset.%s.linseed-{now/s{yyyyMMdd}}-000001>",
				print(tenant, cluster)), config.BootstrapIndexName())

		})
	}
}

func TestBoostrapDomainSetTemplate(t *testing.T) {
	for _, tenant := range []string{backendutils.RandomTenantName(), ""} {
		name := fmt.Sprintf("TestCreateDomainSetTemplate (tenant=%s)", tenant)
		t.Run(name, func(t *testing.T) {

			cluster := backendutils.RandomClusterName()

			expectedTemplate := &Template{
				IndexPatterns: []string{fmt.Sprintf("tigera_secure_ee_threatfeeds_domainnameset.%s.*", print(tenant, cluster))},
				Mappings:      testutils.MustUnmarshalToMap(t, DomainSetMappings),
				Settings:      map[string]interface{}{},
			}

			expectedTemplate.Settings["number_of_shards"] = 1
			expectedTemplate.Settings["number_of_replicas"] = 0

			config := NewTemplateConfig(bapi.DomainNameSet, bapi.ClusterInfo{Cluster: cluster, Tenant: tenant})
			require.Equal(t, fmt.Sprintf("tigera_secure_ee_threatfeeds_domainnameset.%s", print(tenant, cluster)), config.TemplateName())
			template, err := config.Template()
			require.NoError(t, err)
			assertTemplate(t, expectedTemplate, template)

			require.Equal(t, fmt.Sprintf("tigera_secure_ee_threatfeeds_domainnameset.%s.", print(tenant, cluster)), config.Alias())
			require.Equal(t, fmt.Sprintf("<tigera_secure_ee_threatfeeds_domainnameset.%s.linseed-{now/s{yyyyMMdd}}-000001>",
				print(tenant, cluster)), config.BootstrapIndexName())

		})
	}
}

func assertTemplate(t *testing.T, expected *Template, template *Template) {
	require.EqualValues(t, expected.IndexPatterns, template.IndexPatterns)
	require.NotEmpty(t, template.Mappings)
	require.EqualValues(t, expected.Mappings, template.Mappings)
	require.NotEmpty(t, template.Settings)
	require.EqualValues(t, expected.Settings, template.Settings)
}
