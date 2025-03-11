// Copyright (c) 2025 Tigera, Inc. All rights reserved.

package migrator

import (
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/oiler/pkg/config"
	"github.com/projectcalico/calico/oiler/pkg/migrator/operator/audit"
	"github.com/projectcalico/calico/oiler/pkg/migrator/operator/bgp"
	"github.com/projectcalico/calico/oiler/pkg/migrator/operator/compliance"
	"github.com/projectcalico/calico/oiler/pkg/migrator/operator/dns"
	"github.com/projectcalico/calico/oiler/pkg/migrator/operator/events"
	"github.com/projectcalico/calico/oiler/pkg/migrator/operator/flow"
	"github.com/projectcalico/calico/oiler/pkg/migrator/operator/l7"
	"github.com/projectcalico/calico/oiler/pkg/migrator/operator/runtime"
	"github.com/projectcalico/calico/oiler/pkg/migrator/operator/threatfeeds"
	"github.com/projectcalico/calico/oiler/pkg/migrator/operator/waf"
)

func NewAuditEEMigrator(cfg config.Config, primary BackendCatalogue, secondary BackendCatalogue) Migrator[v1.AuditLog] {
	return newAuditMigrator(cfg, v1.AuditLogTypeEE, primary, secondary)
}

func newAuditMigrator(cfg config.Config, auditType v1.AuditLogType, primary BackendCatalogue, secondary BackendCatalogue) Migrator[v1.AuditLog] {
	return Migrator[v1.AuditLog]{
		Primary:   audit.NewOperator(auditType, primary.AuditBackend, primaryClusterInfo(cfg)),
		Secondary: audit.NewOperator(auditType, secondary.AuditBackend, secondaryClusterInfo(cfg)),
		Cfg:       NewConfig(cfg),
	}
}

func primaryClusterInfo(cfg config.Config) bapi.ClusterInfo {
	return bapi.ClusterInfo{Cluster: cfg.PrimaryClusterID, Tenant: cfg.PrimaryTenantID}
}

func secondaryClusterInfo(cfg config.Config) bapi.ClusterInfo {
	return bapi.ClusterInfo{Cluster: cfg.SecondaryClusterID, Tenant: cfg.SecondaryTenantID}
}

func NewAuditKubeMigrator(cfg config.Config, primary BackendCatalogue, secondary BackendCatalogue) Migrator[v1.AuditLog] {
	return newAuditMigrator(cfg, v1.AuditLogTypeKube, primary, secondary)
}

func NewBGPMigrator(cfg config.Config, primary BackendCatalogue, secondary BackendCatalogue) Migrator[v1.BGPLog] {
	return Migrator[v1.BGPLog]{
		Primary:   bgp.NewOperator(primary.BGPBackend, primaryClusterInfo(cfg)),
		Secondary: bgp.NewOperator(secondary.BGPBackend, secondaryClusterInfo(cfg)),
		Cfg:       NewConfig(cfg),
	}
}

func NewBenchmarksMigrator(cfg config.Config, primary BackendCatalogue, secondary BackendCatalogue) Migrator[v1.Benchmarks] {
	return Migrator[v1.Benchmarks]{
		Primary:   compliance.NewBenchmarksOperator(primary.BenchmarksBackend, primaryClusterInfo(cfg)),
		Secondary: compliance.NewBenchmarksOperator(secondary.BenchmarksBackend, secondaryClusterInfo(cfg)),
		Cfg:       NewConfig(cfg),
	}
}

func NewReportsMigrator(cfg config.Config, primary BackendCatalogue, secondary BackendCatalogue) Migrator[v1.ReportData] {
	return Migrator[v1.ReportData]{
		Primary:   compliance.NewReportsOperator(primary.ReportsBackend, primaryClusterInfo(cfg)),
		Secondary: compliance.NewReportsOperator(secondary.ReportsBackend, secondaryClusterInfo(cfg)),
		Cfg:       NewConfig(cfg),
	}
}

func NewSnapshotsMigrator(cfg config.Config, primary BackendCatalogue, secondary BackendCatalogue) Migrator[v1.Snapshot] {
	return Migrator[v1.Snapshot]{
		Primary:   compliance.NewSnapshotsOperator(primary.SnapshotsBackend, primaryClusterInfo(cfg)),
		Secondary: compliance.NewSnapshotsOperator(secondary.SnapshotsBackend, secondaryClusterInfo(cfg)),
		Cfg:       NewConfig(cfg),
	}
}

func NewDNSMigrator(cfg config.Config, primary BackendCatalogue, secondary BackendCatalogue) Migrator[v1.DNSLog] {
	return Migrator[v1.DNSLog]{
		Primary:   dns.NewOperator(primary.DNSLogBackend, primaryClusterInfo(cfg)),
		Secondary: dns.NewOperator(secondary.DNSLogBackend, secondaryClusterInfo(cfg)),
		Cfg:       NewConfig(cfg),
	}
}

func NewEventsMigrator(cfg config.Config, primary BackendCatalogue, secondary BackendCatalogue) Migrator[v1.Event] {
	return Migrator[v1.Event]{
		Primary:   events.NewOperator(primary.EventBackend, primaryClusterInfo(cfg)),
		Secondary: events.NewOperator(secondary.EventBackend, secondaryClusterInfo(cfg)),
		Cfg:       NewConfig(cfg),
	}
}

func NewFlowMigrator(cfg config.Config, primary BackendCatalogue, secondary BackendCatalogue) Migrator[v1.FlowLog] {
	return Migrator[v1.FlowLog]{
		Primary:   flow.NewOperator(primary.FlowLogBackend, primaryClusterInfo(cfg)),
		Secondary: flow.NewOperator(secondary.FlowLogBackend, secondaryClusterInfo(cfg)),
		Cfg:       NewConfig(cfg),
	}
}

func NewL7Migrator(cfg config.Config, primary BackendCatalogue, secondary BackendCatalogue) Migrator[v1.L7Log] {
	return Migrator[v1.L7Log]{
		Primary:   l7.NewOperator(primary.L7LogBackend, primaryClusterInfo(cfg)),
		Secondary: l7.NewOperator(secondary.L7LogBackend, secondaryClusterInfo(cfg)),
		Cfg:       NewConfig(cfg),
	}
}

func NewRuntimeMigrator(cfg config.Config, primary BackendCatalogue, secondary BackendCatalogue) Migrator[v1.RuntimeReport] {
	return Migrator[v1.RuntimeReport]{
		Primary:   runtime.NewOperator(primary.RuntimeBackend, primaryClusterInfo(cfg)),
		Secondary: runtime.NewOperator(secondary.RuntimeBackend, secondaryClusterInfo(cfg)),
		Cfg:       NewConfig(cfg),
	}
}

func NewWAFMigrator(cfg config.Config, primary BackendCatalogue, secondary BackendCatalogue) Migrator[v1.WAFLog] {
	return Migrator[v1.WAFLog]{
		Primary:   waf.NewOperator(primary.WAFBackend, primaryClusterInfo(cfg)),
		Secondary: waf.NewOperator(secondary.WAFBackend, secondaryClusterInfo(cfg)),
		Cfg:       NewConfig(cfg),
	}
}

func NewDomainNameSetMigrator(cfg config.Config, primary BackendCatalogue, secondary BackendCatalogue) Migrator[v1.DomainNameSetThreatFeed] {
	return Migrator[v1.DomainNameSetThreatFeed]{
		Primary:   threatfeeds.NewDomainNameSetOperator(primary.DomainNameSetBackend, primaryClusterInfo(cfg)),
		Secondary: threatfeeds.NewDomainNameSetOperator(secondary.DomainNameSetBackend, secondaryClusterInfo(cfg)),
		Cfg:       NewConfig(cfg),
	}
}

func NewIPSetMigrator(cfg config.Config, primary BackendCatalogue, secondary BackendCatalogue) Migrator[v1.IPSetThreatFeed] {
	return Migrator[v1.IPSetThreatFeed]{
		Primary:   threatfeeds.NewIPSetOperator(primary.IPSetBackend, primaryClusterInfo(cfg)),
		Secondary: threatfeeds.NewIPSetOperator(secondary.IPSetBackend, secondaryClusterInfo(cfg)),
		Cfg:       NewConfig(cfg),
	}
}
