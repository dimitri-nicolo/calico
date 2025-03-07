// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package migrator

import (
	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/linseed/pkg/backend"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	auditbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/audit"
	bgpbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/bgp"
	compliancebackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/compliance"
	dnsbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/dns"
	eventbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/events"
	flowbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/flows"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/index"
	l7backend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/l7"
	runtimebackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/runtime"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	threatfeedsbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/threatfeeds"
	wafbackend "github.com/projectcalico/calico/linseed/pkg/backend/legacy/waf"
	linseed "github.com/projectcalico/calico/linseed/pkg/config"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type BackendCatalogue struct {
	backendType linseed.BackendType

	esClient         lmaelastic.Client
	indexInitializer bapi.IndexInitializer

	AuditBackend         bapi.AuditBackend
	BGPBackend           bapi.BGPBackend
	ReportsBackend       bapi.ReportsBackend
	SnapshotsBackend     bapi.SnapshotsBackend
	BenchmarksBackend    bapi.BenchmarksBackend
	DNSLogBackend        bapi.DNSLogBackend
	FlowLogBackend       bapi.FlowLogBackend
	L7LogBackend         bapi.L7LogBackend
	RuntimeBackend       bapi.RuntimeBackend
	EventBackend         bapi.EventsBackend
	WAFBackend           bapi.WAFBackend
	IPSetBackend         bapi.IPSetBackend
	DomainNameSetBackend bapi.DomainNameSetBackend
}

func MustGetCatalogue(cfg linseed.ElasticClientConfig, backendType linseed.BackendType, logLevel string, source string) BackendCatalogue {
	esClient := backend.MustGetElasticClient(cfg, logLevel, source)

	var auditInitializer bapi.IndexInitializer
	var bgpInitializer bapi.IndexInitializer
	var defaultInitializer bapi.IndexInitializer
	var dnsInitializer bapi.IndexInitializer
	var flowInitializer bapi.IndexInitializer
	var l7Initializer bapi.IndexInitializer

	if backendType == linseed.BackendTypeMultiIndex {
		// Create initializers for indices with special shards / replicas configuration. These initializers
		// will create an index for each new cluster that performs a write requests
		defaultInitializer = templates.NewCachedInitializer(esClient, cfg.ElasticShards, cfg.ElasticReplicas)
		flowInitializer = templates.NewCachedInitializer(esClient, cfg.ElasticFlowShards, cfg.ElasticFlowReplicas)
		dnsInitializer = templates.NewCachedInitializer(esClient, cfg.ElasticDNSShards, cfg.ElasticDNSReplicas)
		l7Initializer = templates.NewCachedInitializer(esClient, cfg.ElasticL7Shards, cfg.ElasticL7Replicas)
		auditInitializer = templates.NewCachedInitializer(esClient, cfg.ElasticAuditShards, cfg.ElasticAuditReplicas)
		bgpInitializer = templates.NewCachedInitializer(esClient, cfg.ElasticBGPShards, cfg.ElasticBGPReplicas)
	} else {
		// Create a no op initializer that will be used for single index setup with index creation disabled.
		// This mode is used to run inside a multi-tenant management cluster for Calico Cloud, where
		// index templates, write aliases and boostrap indices are created via a K8S Job that will
		// be run after provisioning the Elastic Cluster
		noOp := templates.NewNoOpInitializer()
		defaultInitializer = noOp
		flowInitializer = noOp
		dnsInitializer = noOp
		l7Initializer = noOp
		auditInitializer = noOp
		bgpInitializer = noOp
	}
	// Create all the necessary backends.
	var flowLogBackend bapi.FlowLogBackend
	var auditBackend bapi.AuditBackend
	var bgpBackend bapi.BGPBackend
	var reportsBackend bapi.ReportsBackend
	var snapshotsBackend bapi.SnapshotsBackend
	var benchmarksBackend bapi.BenchmarksBackend
	var dnsLogBackend bapi.DNSLogBackend
	var l7LogBackend bapi.L7LogBackend
	var runtimeBackend bapi.RuntimeBackend
	var eventBackend bapi.EventsBackend
	var wafBackend bapi.WAFBackend
	var ipSetBackend bapi.IPSetBackend
	var domainNameSetBackend bapi.DomainNameSetBackend

	maxResultsWindow := cfg.ElasticIndexMaxResultWindow
	switch backendType {
	case linseed.BackendTypeMultiIndex:
		flowLogBackend = flowbackend.NewFlowLogBackend(esClient, flowInitializer, maxResultsWindow)
		auditBackend = auditbackend.NewBackend(esClient, auditInitializer, maxResultsWindow)
		bgpBackend = bgpbackend.NewBackend(esClient, bgpInitializer, maxResultsWindow)
		reportsBackend = compliancebackend.NewReportsBackend(esClient, defaultInitializer, maxResultsWindow)
		snapshotsBackend = compliancebackend.NewSnapshotBackend(esClient, defaultInitializer, maxResultsWindow)
		benchmarksBackend = compliancebackend.NewBenchmarksBackend(esClient, defaultInitializer, maxResultsWindow)
		dnsLogBackend = dnsbackend.NewDNSLogBackend(esClient, dnsInitializer, maxResultsWindow)
		l7LogBackend = l7backend.NewL7LogBackend(esClient, l7Initializer, maxResultsWindow)
		runtimeBackend = runtimebackend.NewBackend(esClient, defaultInitializer, maxResultsWindow)
		eventBackend = eventbackend.NewBackend(esClient, defaultInitializer, maxResultsWindow)
		wafBackend = wafbackend.NewBackend(esClient, defaultInitializer, maxResultsWindow)
		ipSetBackend = threatfeedsbackend.NewIPSetBackend(esClient, defaultInitializer, maxResultsWindow)
		domainNameSetBackend = threatfeedsbackend.NewDomainNameSetBackend(esClient, defaultInitializer, maxResultsWindow)
	case linseed.BackendTypeSingleIndex:
		flowLogBackend = flowbackend.NewSingleIndexFlowLogBackend(esClient, flowInitializer, maxResultsWindow, index.WithBaseIndexName(cfg.ElasticFlowLogsBaseIndexName))
		auditBackend = auditbackend.NewSingleIndexBackend(esClient, auditInitializer, maxResultsWindow, index.WithBaseIndexName(cfg.ElasticAuditLogsBaseIndexName))
		bgpBackend = bgpbackend.NewSingleIndexBackend(esClient, bgpInitializer, maxResultsWindow, index.WithBaseIndexName(cfg.ElasticBGPLogsBaseIndexName))
		reportsBackend = compliancebackend.NewSingleIndexReportsBackend(esClient, defaultInitializer, maxResultsWindow, index.WithBaseIndexName(cfg.ElasticComplianceReportsBaseIndexName))
		snapshotsBackend = compliancebackend.NewSingleIndexSnapshotBackend(esClient, defaultInitializer, maxResultsWindow, index.WithBaseIndexName(cfg.ElasticComplianceSnapshotsBaseIndexName))
		benchmarksBackend = compliancebackend.NewSingleIndexBenchmarksBackend(esClient, defaultInitializer, maxResultsWindow, index.WithBaseIndexName(cfg.ElasticComplianceBenchmarksBaseIndexName))
		dnsLogBackend = dnsbackend.NewSingleIndexDNSLogBackend(esClient, dnsInitializer, maxResultsWindow, index.WithBaseIndexName(cfg.ElasticDNSLogsBaseIndexName))
		l7LogBackend = l7backend.NewSingleIndexL7LogBackend(esClient, l7Initializer, maxResultsWindow, index.WithBaseIndexName(cfg.ElasticL7LogsBaseIndexName))
		runtimeBackend = runtimebackend.NewSingleIndexBackend(esClient, defaultInitializer, maxResultsWindow, index.WithBaseIndexName(cfg.ElasticRuntimeReportsBaseIndexName))
		eventBackend = eventbackend.NewSingleIndexBackend(esClient, defaultInitializer, maxResultsWindow, index.WithBaseIndexName(cfg.ElasticAlertsBaseIndexName))
		wafBackend = wafbackend.NewSingleIndexBackend(esClient, defaultInitializer, maxResultsWindow, index.WithBaseIndexName(cfg.ElasticWAFLogsBaseIndexName))
		ipSetBackend = threatfeedsbackend.NewSingleIndexIPSetBackend(esClient, defaultInitializer, maxResultsWindow, index.WithBaseIndexName(cfg.ElasticThreatFeedsIPSetBaseIndexName))
		domainNameSetBackend = threatfeedsbackend.NewSingleIndexDomainNameSetBackend(esClient, defaultInitializer, maxResultsWindow, index.WithBaseIndexName(cfg.ElasticThreatFeedsDomainSetBaseIndexName))
	default:
		logrus.Fatalf("Invalid backend type: %s", backendType)

	}

	return BackendCatalogue{
		esClient:             esClient,
		backendType:          backendType,
		indexInitializer:     flowInitializer,
		FlowLogBackend:       flowLogBackend,
		AuditBackend:         auditBackend,
		BGPBackend:           bgpBackend,
		ReportsBackend:       reportsBackend,
		SnapshotsBackend:     snapshotsBackend,
		BenchmarksBackend:    benchmarksBackend,
		DNSLogBackend:        dnsLogBackend,
		L7LogBackend:         l7LogBackend,
		RuntimeBackend:       runtimeBackend,
		EventBackend:         eventBackend,
		IPSetBackend:         ipSetBackend,
		DomainNameSetBackend: domainNameSetBackend,
		WAFBackend:           wafBackend,
	}
}
