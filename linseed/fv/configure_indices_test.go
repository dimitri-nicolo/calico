// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/index"
	"github.com/projectcalico/calico/linseed/pkg/backend/testutils"
	"github.com/projectcalico/calico/linseed/pkg/config"
)

// configureIndicesSetupAndTeardown performs additional setup and teardown for ingestion tests.
func configureIndicesSetupAndTeardown(t *testing.T, idx bapi.Index) func() {
	// Hook logrus into testing.T
	config.ConfigureLogging("DEBUG")
	logCancel := logutils.RedirectLogrusToTestingT(t)

	// Create an ES client.
	var err error
	esClient, err = elastic.NewSimpleClient(elastic.SetURL("http://localhost:9200"), elastic.SetInfoLog(logrus.StandardLogger()))
	require.NoError(t, err)

	// Set up context with a timeout.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)

	return func() {
		_ = testutils.CleanupIndices(context.Background(), esClient, idx.IsSingleIndex(), idx, clusterInfo)
		logCancel()
		cancel()
	}
}

func TestFV_ConfigureFlowIndices(t *testing.T) {
	alertsIndex := index.AlertsIndex(index.WithBaseIndexName("calico_alerts_free"), index.WithILMPolicyName("calico_free"))
	auditIndex := index.AuditLogIndex(index.WithBaseIndexName("calico_auditlogs_free"), index.WithILMPolicyName("calico_free"))
	bgpIndex := index.BGPLogIndex(index.WithBaseIndexName("calico_bgplogs_free"), index.WithILMPolicyName("calico_free"))
	dnsIndex := index.DNSLogIndex(index.WithBaseIndexName("calico_dnslogs_free"), index.WithILMPolicyName("calico_free"))
	flowIndex := index.FlowLogIndex(index.WithBaseIndexName("calico_flowlogs_free"), index.WithILMPolicyName("calico_free"))
	complianceBenchmarksIndex := index.ComplianceBenchmarksIndex(index.WithBaseIndexName("calico_compliance_benchmarks_free"), index.WithILMPolicyName("calico_free"))
	complianceReportsIndex := index.ComplianceReportsIndex(index.WithBaseIndexName("calico_compliance_reports_free"), index.WithILMPolicyName("calico_free"))
	complianceSnapshotsIndex := index.ComplianceSnapshotsIndex(index.WithBaseIndexName("calico_compliance_snapshots_free"), index.WithILMPolicyName("calico_free"))
	l7Index := index.L7LogIndex(index.WithBaseIndexName("calico_l7logs_free"), index.WithILMPolicyName("calico_free"))
	runtimeIndex := index.RuntimeReportsIndex(index.WithBaseIndexName("calico_runtime_reports_free"), index.WithILMPolicyName("calico_free"))
	threatFeedsIPSetIndex := index.ThreatFeedsIPSetIndex(index.WithBaseIndexName("calico_thread_feeds_ip_set_free"), index.WithILMPolicyName("calico_free"))
	threatFeedsDomainSetIndex := index.ThreatFeedsDomainSetIndex(index.WithBaseIndexName("calico_thread_feeds_domain_set_free"), index.WithILMPolicyName("calico_free"))
	wafIndex := index.WAFLogIndex(index.WithBaseIndexName("calico_waf_logs_free"), index.WithILMPolicyName("calico_free"))

	var tests = []struct {
		index       bapi.Index
		linseedArgs *RunConfigureElasticArgs
	}{
		{
			index: alertsIndex,
			linseedArgs: &RunConfigureElasticArgs{
				AlertBaseIndexName: alertsIndex.Name(bapi.ClusterInfo{}),
				AlertPolicyName:    alertsIndex.ILMPolicyName(),
			},
		},
		{
			index: auditIndex,
			linseedArgs: &RunConfigureElasticArgs{
				AuditBaseIndexName: auditIndex.Name(bapi.ClusterInfo{}),
				AuditPolicyName:    auditIndex.ILMPolicyName(),
			},
		},
		{
			index: bgpIndex,
			linseedArgs: &RunConfigureElasticArgs{
				BGPBaseIndexName: bgpIndex.Name(bapi.ClusterInfo{}),
				BGPPolicyName:    bgpIndex.ILMPolicyName(),
			},
		},
		{
			index: dnsIndex,
			linseedArgs: &RunConfigureElasticArgs{
				DNSBaseIndexName: dnsIndex.Name(bapi.ClusterInfo{}),
				DNSPolicyName:    dnsIndex.ILMPolicyName(),
			},
		},
		{
			index: complianceBenchmarksIndex,
			linseedArgs: &RunConfigureElasticArgs{
				ComplianceBenchmarksBaseIndexName: complianceBenchmarksIndex.Name(bapi.ClusterInfo{}),
				ComplianceBenchmarksPolicyName:    complianceBenchmarksIndex.ILMPolicyName(),
			},
		},
		{
			index: complianceReportsIndex,
			linseedArgs: &RunConfigureElasticArgs{
				ComplianceReportsBaseIndexName: complianceReportsIndex.Name(bapi.ClusterInfo{}),
				ComplianceReportsPolicyName:    complianceReportsIndex.ILMPolicyName(),
			},
		},
		{
			index: complianceSnapshotsIndex,
			linseedArgs: &RunConfigureElasticArgs{
				ComplianceSnapshotsBaseIndexName: complianceSnapshotsIndex.Name(bapi.ClusterInfo{}),
				ComplianceSnapshotsPolicyName:    complianceSnapshotsIndex.ILMPolicyName(),
			},
		},
		{
			index: flowIndex,
			linseedArgs: &RunConfigureElasticArgs{
				FlowBaseIndexName: flowIndex.Name(bapi.ClusterInfo{}),
				FlowPolicyName:    flowIndex.ILMPolicyName(),
			},
		},
		{
			index: l7Index,
			linseedArgs: &RunConfigureElasticArgs{
				L7BaseIndexName: l7Index.Name(bapi.ClusterInfo{}),
				L7PolicyName:    l7Index.ILMPolicyName(),
			},
		},
		{
			index: runtimeIndex,
			linseedArgs: &RunConfigureElasticArgs{
				RuntimeReportsBaseIndexName: runtimeIndex.Name(bapi.ClusterInfo{}),
				RuntimeReportsPolicyName:    runtimeIndex.ILMPolicyName(),
			},
		},
		{
			index: threatFeedsDomainSetIndex,
			linseedArgs: &RunConfigureElasticArgs{
				ThreatFeedsDomainSetBaseIndexName: threatFeedsDomainSetIndex.Name(bapi.ClusterInfo{}),
				ThreatFeedsDomainSetPolicyName:    threatFeedsDomainSetIndex.ILMPolicyName(),
			},
		},
		{
			index: threatFeedsIPSetIndex,
			linseedArgs: &RunConfigureElasticArgs{
				ThreatFeedsIPSetBaseIndexName: threatFeedsIPSetIndex.Name(bapi.ClusterInfo{}),
				ThreatFeedsIPSetPolicyName:    threatFeedsIPSetIndex.ILMPolicyName(),
			},
		},
		{
			index: wafIndex,
			linseedArgs: &RunConfigureElasticArgs{
				WAFBaseIndexName: wafIndex.Name(bapi.ClusterInfo{}),
				WAFPolicyName:    wafIndex.ILMPolicyName(),
			},
		},
	}

	for _, tt := range tests {

		t.Run(fmt.Sprintf("Configure Elastic Indices %s [SingleIndex]", tt.index.DataType()), func(t *testing.T) {
			defer configureIndicesSetupAndTeardown(t, tt.index)()

			// Start a linseed configuration instance.
			linseed := RunConfigureElasticLinseed(t, tt.linseedArgs)
			defer func() {
				if linseed.ListedInDockerPS() {
					linseed.Stop()
				}
			}()

			testutils.CheckSingleIndexTemplateBootstrapping(t, ctx, esClient, tt.index, bapi.ClusterInfo{})
		})
	}
}
