// Copyright (c) 2018-2023 Tigera, Inc. All rights reserved.

package collector

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/felix/calc"
	"github.com/projectcalico/calico/felix/collector/dnslog"
	"github.com/projectcalico/calico/felix/collector/flowlog"
	"github.com/projectcalico/calico/felix/collector/l7log"
	p "github.com/projectcalico/calico/felix/collector/prometheus"
	"github.com/projectcalico/calico/felix/collector/reporter"
	"github.com/projectcalico/calico/felix/collector/reporter/file"
	"github.com/projectcalico/calico/felix/config"
	"github.com/projectcalico/calico/felix/rules"
	"github.com/projectcalico/calico/felix/wireguard"
	"github.com/projectcalico/calico/libcalico-go/lib/health"
)

const (
	// Log dispatcher names
	FlowLogsFileDispatcherName = "file"
	DNSLogsFileDispatcherName  = "dnsfile"
	L7LogsFileDispatcherName   = "l7file"
)

// New creates the required dataplane stats collector, reporters and aggregators.
// Returns a collector that statistics should be reported to.
func New(
	configParams *config.Config,
	lookupsCache *calc.LookupsCache,
	healthAggregator *health.HealthAggregator,
) Collector {
	registry := prometheus.NewRegistry()

	if configParams.WireguardEnabled {
		registry.MustRegister(wireguard.MustNewWireguardMetrics())
	}

	rm := reporter.NewManager(configParams.FlowLogsCollectorDebugTrace)
	if configParams.PrometheusReporterEnabled {
		fipsModeEnabled := os.Getenv("FIPS_MODE_ENABLED") == "true"
		log.WithFields(log.Fields{
			"port":            configParams.PrometheusReporterPort,
			"fipsModeEnabled": fipsModeEnabled,
			"certFile":        configParams.PrometheusReporterCertFile,
			"keyFile":         configParams.PrometheusReporterKeyFile,
			"caFile":          configParams.PrometheusReporterCAFile,
		}).Info("Starting prometheus reporter")

		pr := p.NewReporter(
			registry,
			configParams.PrometheusReporterPort,
			configParams.DeletedMetricsRetentionSecs,
			configParams.PrometheusReporterCertFile,
			configParams.PrometheusReporterKeyFile,
			configParams.PrometheusReporterCAFile,
			fipsModeEnabled,
		)
		pr.AddAggregator(p.NewPolicyRulesAggregator(configParams.DeletedMetricsRetentionSecs, configParams.FelixHostname))
		pr.AddAggregator(p.NewDeniedPacketsAggregator(configParams.DeletedMetricsRetentionSecs, configParams.FelixHostname))
		rm.RegisterMetricsReporter(pr)
	}
	dispatchers := map[string]reporter.LogDispatcher{}
	if configParams.FlowLogsFileEnabled {
		log.WithFields(log.Fields{
			"directory": configParams.GetFlowLogsFileDirectory(),
			"max_size":  configParams.FlowLogsFileMaxFileSizeMB,
			"max_files": configParams.FlowLogsFileMaxFiles,
		}).Info("Creating Flow Logs FileDispatcher")
		fd := file.NewDispatcher(
			configParams.GetFlowLogsFileDirectory(),
			file.FlowLogFilename,
			configParams.FlowLogsFileMaxFileSizeMB,
			configParams.FlowLogsFileMaxFiles,
		)
		dispatchers[FlowLogsFileDispatcherName] = fd
	}
	if len(dispatchers) > 0 {
		log.Info("Creating Flow Logs Reporter")
		var offsetReader flowlog.LogOffset = &flowlog.NoOpLogOffset{}
		if configParams.FlowLogsDynamicAggregationEnabled {
			offsetReader = flowlog.NewRangeLogOffset(flowlog.NewFluentDLogOffsetReader(configParams.GetFlowLogsPositionFilePath()),
				int64(configParams.FlowLogsAggregationThresholdBytes))
		}
		cw := flowlog.NewReporter(dispatchers, configParams.FlowLogsFlushInterval, healthAggregator,
			configParams.FlowLogsEnableHostEndpoint, configParams.FlowLogsCollectorDebugTrace, offsetReader)
		configureFlowAggregation(configParams, cw)
		rm.RegisterMetricsReporter(cw)
	}

	syslogReporter := reporter.NewSyslog(configParams.SyslogReporterNetwork, configParams.SyslogReporterAddress)
	if syslogReporter != nil {
		rm.RegisterMetricsReporter(syslogReporter)
	}
	rm.Start()
	statsCollector := newCollector(
		lookupsCache,
		rm,
		&Config{
			StatsDumpFilePath:            configParams.GetStatsDumpFilePath(),
			AgeTimeout:                   config.DefaultAgeTimeout,
			InitialReportingDelay:        config.DefaultInitialReportingDelay,
			ExportingInterval:            config.DefaultExportingInterval,
			EnableServices:               configParams.FlowLogsFileIncludeService,
			EnableNetworkSets:            configParams.FlowLogsEnableNetworkSets,
			MaxOriginalSourceIPsIncluded: configParams.FlowLogsMaxOriginalIPsIncluded,
			IsBPFDataplane:               configParams.BPFEnabled,
		},
	)

	if configParams.DNSLogsFileEnabled {
		// Create the reporter, aggregator and dispatcher for DNS logging.
		dnsLogReporter := dnslog.NewReporter(
			map[string]reporter.LogDispatcher{
				DNSLogsFileDispatcherName: file.NewDispatcher(
					configParams.DNSLogsFileDirectory,
					file.DNSLogFilename,
					configParams.DNSLogsFileMaxFileSizeMB,
					configParams.DNSLogsFileMaxFiles,
				),
			},
			configParams.DNSLogsFlushInterval,
			healthAggregator,
		)
		dnsLogReporter.AddAggregator(
			dnslog.NewAggregator().
				AggregateOver(dnslog.AggregationKind(configParams.DNSLogsFileAggregationKind)).
				IncludeLabels(configParams.DNSLogsFileIncludeLabels).
				PerNodeLimit(configParams.DNSLogsFilePerNodeLimit),
			[]string{DNSLogsFileDispatcherName},
		)
		statsCollector.SetDNSLogReporter(dnsLogReporter)
	}

	if configParams.L7LogsFileEnabled {
		// Create the reporter, aggregator and dispatcher for L7 logging.
		l7LogReporter := l7log.NewReporter(
			map[string]reporter.LogDispatcher{
				L7LogsFileDispatcherName: file.NewDispatcher(
					configParams.L7LogsFileDirectory,
					file.L7LogFilename,
					configParams.L7LogsFileMaxFileSizeMB,
					configParams.L7LogsFileMaxFiles,
				),
			},
			configParams.L7LogsFlushInterval,
			healthAggregator,
		)
		// Create the aggregation kind
		aggKind := l7log.AggregationKindFromConfigParams(configParams)
		l7LogReporter.AddAggregator(
			l7log.NewAggregator().
				AggregateOver(aggKind).
				PerNodeLimit(configParams.L7LogsFilePerNodeLimit),
			[]string{L7LogsFileDispatcherName},
		)
		statsCollector.SetL7LogReporter(l7LogReporter)
	}

	return statsCollector
}

// configureFlowAggregation adds appropriate aggregators to the FlowLogReporter, depending on configuration.
func configureFlowAggregation(configParams *config.Config, fr *flowlog.FlowLogReporter) {
	addedFileAllow := false
	addedFileDeny := false
	if configParams.FlowLogsFileEnabled {
		if !addedFileAllow && configParams.FlowLogsFileEnabledForAllowed {
			log.Info("Creating Flow Logs Aggregator for allowed")
			caa := flowlog.NewAggregator().
				AggregateOver(flowlog.AggregationKind(configParams.FlowLogsFileAggregationKindForAllowed)).
				DisplayDebugTraceLogs(configParams.FlowLogsCollectorDebugTrace).
				IncludeLabels(configParams.FlowLogsFileIncludeLabels).
				IncludePolicies(configParams.FlowLogsFileIncludePolicies).
				IncludeService(configParams.FlowLogsFileIncludeService).
				IncludeProcess(configParams.FlowLogsCollectProcessInfo).
				IncludeTcpStats(configParams.FlowLogsCollectTcpStats).
				MaxOriginalIPsSize(configParams.FlowLogsMaxOriginalIPsIncluded).
				MaxDomains(configParams.FlowLogsFileDomainsLimit).
				PerFlowProcessLimit(configParams.FlowLogsFilePerFlowProcessLimit).
				PerFlowProcessArgsLimit(configParams.FlowLogsFilePerFlowProcessArgsLimit).
				NatOutgoingPortLimit(configParams.FlowLogsFileNatOutgoingPortLimit).
				ForAction(rules.RuleActionAllow)
			log.Info("Adding Flow Logs Aggregator (allowed) for File logs")
			fr.AddAggregator(caa, []string{FlowLogsFileDispatcherName})
		}
		if !addedFileDeny && configParams.FlowLogsFileEnabledForDenied {
			log.Info("Creating Flow Logs Aggregator for denied")
			cad := flowlog.NewAggregator().
				AggregateOver(flowlog.AggregationKind(configParams.FlowLogsFileAggregationKindForDenied)).
				DisplayDebugTraceLogs(configParams.FlowLogsCollectorDebugTrace).
				IncludeLabels(configParams.FlowLogsFileIncludeLabels).
				IncludePolicies(configParams.FlowLogsFileIncludePolicies).
				IncludeService(configParams.FlowLogsFileIncludeService).
				IncludeTcpStats(configParams.FlowLogsCollectTcpStats).
				IncludeProcess(configParams.FlowLogsCollectProcessInfo).
				MaxOriginalIPsSize(configParams.FlowLogsMaxOriginalIPsIncluded).
				MaxDomains(configParams.FlowLogsFileDomainsLimit).
				PerFlowProcessLimit(configParams.FlowLogsFilePerFlowProcessLimit).
				PerFlowProcessArgsLimit(configParams.FlowLogsFilePerFlowProcessArgsLimit).
				NatOutgoingPortLimit(configParams.FlowLogsFileNatOutgoingPortLimit).
				ForAction(rules.RuleActionDeny)
			log.Info("Adding Flow Logs Aggregator (denied) for File logs")
			fr.AddAggregator(cad, []string{FlowLogsFileDispatcherName})
		}
	}
}
