// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package main

import (
	"context"
	"os"
	"time"

	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/linseed/pkg/backend"
	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/index"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	"github.com/projectcalico/calico/linseed/pkg/config"
)

func boostrapElasticIndices() {
	// Read and reconcile configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}
	if cfg.Backend != config.BackendTypeSingleIndex {
		panic("Invalid configuration. Configuration job needs to run in single index mode")
	}

	// Configure logging
	config.ConfigureLogging(cfg.LogLevel)
	logrus.Debugf("Starting with %#v", cfg)

	esClient := backend.MustGetElasticClient(*cfg)
	createSingleIndexIndices(cfg, esClient)

	os.Exit(0)
}

func createSingleIndexIndices(cfg *config.Config, esClient lmaelastic.Client) {
	// We are only configuring indices and there is no need to start the HTTP server
	logrus.Info("Configuring Elastic indices")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ch := make(chan struct{}, 1)

	go func() {
		// Create template caches for indices with special shards / replicas configuration
		defaultInitializer := templates.NewCachedInitializer(esClient, cfg.ElasticShards, cfg.ElasticReplicas)
		flowInitializer := templates.NewCachedInitializer(esClient, cfg.ElasticFlowShards, cfg.ElasticFlowReplicas)
		dnsInitializer := templates.NewCachedInitializer(esClient, cfg.ElasticDNSShards, cfg.ElasticDNSReplicas)
		l7Initializer := templates.NewCachedInitializer(esClient, cfg.ElasticL7Shards, cfg.ElasticL7Replicas)
		auditInitializer := templates.NewCachedInitializer(esClient, cfg.ElasticAuditShards, cfg.ElasticAuditReplicas)
		bgpInitializer := templates.NewCachedInitializer(esClient, cfg.ElasticBGPShards, cfg.ElasticBGPReplicas)

		// Create all indices with the given configurations (name and ilm policy)
		alertIndex := index.AlertsIndex(index.WithIndexName(cfg.ElasticAlertsIndexName), index.WithPolicyName(cfg.ElasticAlertsPolicyName))
		auditIndex := index.AuditLogIndex(index.WithIndexName(cfg.ElasticAuditLogsIndexName), index.WithPolicyName(cfg.ElasticAuditLogsPolicyName))
		bgpIndex := index.BGPLogIndex(index.WithIndexName(cfg.ElasticBGPLogsIndexName), index.WithPolicyName(cfg.ElasticBGPLogsPolicyName))
		dnsIndex := index.DNSLogIndex(index.WithIndexName(cfg.ElasticDNSLogsIndexName), index.WithPolicyName(cfg.ElasticDNSLogsPolicyName))
		flowIndex := index.FlowLogIndex(index.WithIndexName(cfg.ElasticFlowLogsIndexName), index.WithPolicyName(cfg.ElasticFlowLogsPolicyName))
		complianceBenchmarksIndex := index.ComplianceBenchmarksIndex(index.WithIndexName(cfg.ElasticComplianceBenchmarksIndexName), index.WithPolicyName(cfg.ElasticComplianceBenchmarksPolicyName))
		complianceReportsIndex := index.ComplianceReportsIndex(index.WithIndexName(cfg.ElasticComplianceReportsIndexName), index.WithPolicyName(cfg.ElasticComplianceReportsPolicyName))
		complianceSnapshotsIndex := index.ComplianceSnapshotsIndex(index.WithIndexName(cfg.ElasticComplianceSnapshotsIndexName), index.WithPolicyName(cfg.ElasticComplianceSnapshotsPolicyName))
		l7Index := index.L7LogIndex(index.WithIndexName(cfg.ElasticL7LogsIndexName), index.WithPolicyName(cfg.ElasticL7LogsPolicyName))
		runtimeIndex := index.RuntimeReportsIndex(index.WithIndexName(cfg.ElasticRuntimeReportsIndexName), index.WithPolicyName(cfg.ElasticRuntimeReportsPolicyName))
		threatFeedsIPSetIndex := index.ThreatFeedsIPSetIndex(index.WithIndexName(cfg.ElasticThreatFeedsIPSetIndexName), index.WithPolicyName(cfg.ElasticThreatFeedsIPSetIPolicyName))
		threatFeedsDomainSetIndex := index.ThreatFeedsDomainSetIndex(index.WithIndexName(cfg.ElasticThreatFeedsDomainNameSetIndexName), index.WithPolicyName(cfg.ElasticThreatFeedsDomainNameSetPolicyName))
		wafIndex := index.WAFLogIndex(index.WithIndexName(cfg.ElasticWAFLogsIndexName), index.WithPolicyName(cfg.ElasticWAFLogsPolicyName))

		// Indices defined below share the same configuration for shards / replicas
		indices := []api.Index{alertIndex, complianceBenchmarksIndex, complianceReportsIndex, complianceSnapshotsIndex,
			runtimeIndex, threatFeedsIPSetIndex, threatFeedsDomainSetIndex, wafIndex}
		for _, idx := range indices {
			configureIndex(idx, defaultInitializer, ctx)
		}
		// Indices below can have replicas / shards user configured
		configureIndex(auditIndex, auditInitializer, ctx)
		configureIndex(bgpIndex, bgpInitializer, ctx)
		configureIndex(dnsIndex, dnsInitializer, ctx)
		configureIndex(flowIndex, flowInitializer, ctx)
		configureIndex(l7Index, l7Initializer, ctx)
		ch <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		logrus.Fatal("Indices configuration time out")
	case <-ch:
		logrus.Info("Finished configuring Elastic indices")
	}
}

func configureIndex(idx api.Index, cache api.IndexInitializer, ctx context.Context) {
	var emptyClusterInfo api.ClusterInfo
	indexName := idx.Name(emptyClusterInfo)
	policyName := idx.ILMPolicyName()
	if len(indexName) == 0 {
		logrus.Warnf("Skipping index configuration as no name was provided for data type %s", idx.DataType())
		return
	}
	if len(policyName) == 0 {
		logrus.Warnf("Skipping index configuration as no policy name was provided for data type %s", idx.DataType())
		return
	}

	logrus.Infof("Configure index %s for data type %s", indexName, idx.DataType())
	err := cache.Initialize(ctx, idx, emptyClusterInfo)
	if err != nil {
		logrus.WithError(err).Fatalf("Failed to configure elastic index %s", indexName)
	}
}
