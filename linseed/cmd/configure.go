// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package main

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/linseed/pkg/backend"
	"github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/index"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	"github.com/projectcalico/calico/linseed/pkg/config"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
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

type indexInitializer struct {
	index       api.Index
	initializer api.IndexInitializer
}

func createSingleIndexIndices(cfg *config.Config, esClient lmaelastic.Client) {
	// We are only configuring indices and there is no need to start the HTTP server
	logrus.Info("Configuring Elastic indices")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create template caches for indices with special shards / replicas configuration
	defaultInitializer := templates.NewCachedInitializer(esClient, cfg.ElasticShards, cfg.ElasticReplicas)
	flowInitializer := templates.NewCachedInitializer(esClient, cfg.ElasticFlowShards, cfg.ElasticFlowReplicas)
	dnsInitializer := templates.NewCachedInitializer(esClient, cfg.ElasticDNSShards, cfg.ElasticDNSReplicas)
	l7Initializer := templates.NewCachedInitializer(esClient, cfg.ElasticL7Shards, cfg.ElasticL7Replicas)
	auditInitializer := templates.NewCachedInitializer(esClient, cfg.ElasticAuditShards, cfg.ElasticAuditReplicas)
	bgpInitializer := templates.NewCachedInitializer(esClient, cfg.ElasticBGPShards, cfg.ElasticBGPReplicas)

	// Create all indices with the given configurations (name and ilm policy)
	alertIndex := index.AlertsIndex(index.WithBaseIndexName(cfg.ElasticAlertsBaseIndexName), index.WithILMPolicyName(cfg.ElasticAlertsPolicyName))
	auditIndex := index.AuditLogIndex(index.WithBaseIndexName(cfg.ElasticAuditLogsBaseIndexName), index.WithILMPolicyName(cfg.ElasticAuditLogsPolicyName))
	bgpIndex := index.BGPLogIndex(index.WithBaseIndexName(cfg.ElasticBGPLogsBaseIndexName), index.WithILMPolicyName(cfg.ElasticBGPLogsPolicyName))
	dnsIndex := index.DNSLogIndex(index.WithBaseIndexName(cfg.ElasticDNSLogsBaseIndexName), index.WithILMPolicyName(cfg.ElasticDNSLogsPolicyName))
	flowIndex := index.FlowLogIndex(index.WithBaseIndexName(cfg.ElasticFlowLogsBaseIndexName), index.WithILMPolicyName(cfg.ElasticFlowLogsPolicyName))
	complianceBenchmarksIndex := index.ComplianceBenchmarksIndex(index.WithBaseIndexName(cfg.ElasticComplianceBenchmarksBaseIndexName), index.WithILMPolicyName(cfg.ElasticComplianceBenchmarksPolicyName))
	complianceReportsIndex := index.ComplianceReportsIndex(index.WithBaseIndexName(cfg.ElasticComplianceReportsBaseIndexName), index.WithILMPolicyName(cfg.ElasticComplianceReportsPolicyName))
	complianceSnapshotsIndex := index.ComplianceSnapshotsIndex(index.WithBaseIndexName(cfg.ElasticComplianceSnapshotsBaseIndexName), index.WithILMPolicyName(cfg.ElasticComplianceSnapshotsPolicyName))
	l7Index := index.L7LogIndex(index.WithBaseIndexName(cfg.ElasticL7LogsBaseIndexName), index.WithILMPolicyName(cfg.ElasticL7LogsPolicyName))
	runtimeIndex := index.RuntimeReportsIndex(index.WithBaseIndexName(cfg.ElasticRuntimeReportsBaseIndexName), index.WithILMPolicyName(cfg.ElasticRuntimeReportsPolicyName))
	threatFeedsIPSetIndex := index.ThreatFeedsIPSetIndex(index.WithBaseIndexName(cfg.ElasticThreatFeedsIPSetBaseIndexName), index.WithILMPolicyName(cfg.ElasticThreatFeedsIPSetIPolicyName))
	threatFeedsDomainSetIndex := index.ThreatFeedsDomainSetIndex(index.WithBaseIndexName(cfg.ElasticThreatFeedsDomainSetBaseIndexName), index.WithILMPolicyName(cfg.ElasticThreatFeedsDomainSetPolicyName))
	wafIndex := index.WAFLogIndex(index.WithBaseIndexName(cfg.ElasticWAFLogsBaseIndexName), index.WithILMPolicyName(cfg.ElasticWAFLogsPolicyName))

	initialization := []indexInitializer{
		// Indices defined below share the same configuration for shards / replicas
		{index: alertIndex, initializer: defaultInitializer},
		{index: complianceBenchmarksIndex, initializer: defaultInitializer},
		{index: complianceReportsIndex, initializer: defaultInitializer},
		{index: complianceSnapshotsIndex, initializer: defaultInitializer},
		{index: runtimeIndex, initializer: defaultInitializer},
		{index: threatFeedsDomainSetIndex, initializer: defaultInitializer},
		{index: threatFeedsIPSetIndex, initializer: defaultInitializer},
		{index: wafIndex, initializer: defaultInitializer},
		// Indices below can have replicas / shards user configured
		{index: auditIndex, initializer: auditInitializer},
		{index: bgpIndex, initializer: bgpInitializer},
		{index: dnsIndex, initializer: dnsInitializer},
		{index: flowIndex, initializer: flowInitializer},
		{index: l7Index, initializer: l7Initializer},
	}

	for _, idx := range initialization {
		configureIndex(ctx, idx.index, idx.initializer)
	}

	logrus.Info("Finished configuring Elastic indices")
}

func configureIndex(ctx context.Context, idx api.Index, cache api.IndexInitializer) {
	var emptyClusterInfo api.ClusterInfo
	indexName := idx.Name(emptyClusterInfo)
	policyName := idx.ILMPolicyName()
	if len(indexName) == 0 {
		logrus.Warnf("Skipping index configuration as no name was provided for data type %s", idx.DataType())
		return
	}
	if idx.HasLifecycleEnabled() && len(policyName) == 0 {
		logrus.Warnf("Skipping index configuration as no policy name was provided for data type %s", idx.DataType())
		return
	}

	logrus.Infof("Configure index %s for data type %s", indexName, idx.DataType())
	err := cache.Initialize(ctx, idx, emptyClusterInfo)
	if err != nil {
		logrus.WithError(err).Fatalf("Failed to configure elastic index %s", indexName)
	}
}
