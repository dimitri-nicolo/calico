// Copyright (c) 2018-2021 Tigera, Inc. All rights reserved.

package collector

import (
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"

	"github.com/projectcalico/felix/calc"
	"github.com/projectcalico/felix/collector/testutil"
	"github.com/projectcalico/felix/config"
	"github.com/projectcalico/felix/rules"
	"github.com/projectcalico/libcalico-go/lib/health"
)

const (
	// Log dispatcher names
	CloudWatchLogsDispatcherName = "cloudwatch"
	FlowLogsFileDispatcherName   = "file"
	DNSLogsFileDispatcherName    = "dnsfile"
	L7LogsFileDispatcherName     = "l7file"
)

// New creates the required dataplane stats collector, reporters and aggregators.
// Returns a collector that statistics should be reported to.
func New(
	configParams *config.Config,
	lookupsCache *calc.LookupsCache,
	healthAggregator *health.HealthAggregator,
) Collector {

	rm := NewReporterManager()
	if configParams.PrometheusReporterEnabled {
		pr := NewPrometheusReporter(configParams.PrometheusReporterPort,
			configParams.DeletedMetricsRetentionSecs,
			configParams.PrometheusReporterCertFile,
			configParams.PrometheusReporterKeyFile,
			configParams.PrometheusReporterCAFile)
		pr.AddAggregator(NewPolicyRulesAggregator(configParams.DeletedMetricsRetentionSecs, configParams.FelixHostname))
		pr.AddAggregator(NewDeniedPacketsAggregator(configParams.DeletedMetricsRetentionSecs, configParams.FelixHostname))
		rm.RegisterMetricsReporter(pr)
	}
	log.Debugf("CloudWatchLogsReporterEnabled %v", configParams.CloudWatchLogsReporterEnabled)
	dispatchers := map[string]LogDispatcher{}
	if configParams.CloudWatchLogsReporterEnabled {
		logGroupName := strings.Replace(
			configParams.CloudWatchLogsLogGroupName,
			"<cluster-guid>",
			configParams.ClusterGUID,
			1,
		)
		logStreamName := strings.Replace(
			configParams.CloudWatchLogsLogStreamName,
			"<felix-hostname>",
			configParams.FelixHostname,
			1,
		)
		var cwl cloudwatchlogsiface.CloudWatchLogsAPI
		if configParams.DebugCloudWatchLogsFile != "" {
			log.Info("Creating Debug CloudWatchLogsAPI")
			// Allow CloudWatch logging to be FV tested without incurring AWS
			// costs, by calling a mock AWS API instead of the real one.
			cwl = testutil.NewDebugCloudWatchLogsFile(logGroupName, configParams.DebugCloudWatchLogsFile)
		}
		log.Info("Creating Flow Logs CloudWatchDispatcher")
		cwd := NewCloudWatchDispatcher(logGroupName, logStreamName, configParams.CloudWatchLogsRetentionDays, cwl)
		dispatchers[CloudWatchLogsDispatcherName] = cwd
	}
	if configParams.FlowLogsFileEnabled {
		log.WithFields(log.Fields{
			"directory": configParams.GetFlowLogsFileDirectory(),
			"max_size":  configParams.FlowLogsFileMaxFileSizeMB,
			"max_files": configParams.FlowLogsFileMaxFiles,
		}).Info("Creating Flow Logs FileDispatcher")
		fd := NewFileDispatcher(
			configParams.GetFlowLogsFileDirectory(),
			FlowLogFilename,
			configParams.FlowLogsFileMaxFileSizeMB,
			configParams.FlowLogsFileMaxFiles,
		)
		dispatchers[FlowLogsFileDispatcherName] = fd
	}
	if len(dispatchers) > 0 {
		log.Info("Creating Flow Logs Reporter")
		var offsetReader LogOffset = &NoOpLogOffset{}
		if configParams.FlowLogsDynamicAggregationEnabled {
			offsetReader = NewRangeLogOffset(NewFluentDLogOffsetReader(configParams.GetFlowLogsPositionFilePath()),
				int64(configParams.FlowLogsAggregationThresholdBytes))
		}
		cw := NewFlowLogsReporter(dispatchers, configParams.FlowLogsFlushInterval, healthAggregator,
			configParams.FlowLogsEnableHostEndpoint, offsetReader)
		configureFlowAggregation(configParams, cw)
		rm.RegisterMetricsReporter(cw)
	}

	if configParams.CloudWatchMetricsReporterEnabled {
		cwm := NewCloudWatchMetricsReporter(configParams.CloudWatchMetricsPushIntervalSecs, configParams.ClusterGUID)
		rm.RegisterMetricsReporter(cwm)
	}

	syslogReporter := NewSyslogReporter(configParams.SyslogReporterNetwork, configParams.SyslogReporterAddress)
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
		dnsLogReporter := NewDNSLogReporter(
			map[string]LogDispatcher{
				DNSLogsFileDispatcherName: NewFileDispatcher(
					configParams.DNSLogsFileDirectory,
					DNSLogFilename,
					configParams.DNSLogsFileMaxFileSizeMB,
					configParams.DNSLogsFileMaxFiles,
				),
			},
			configParams.DNSLogsFlushInterval,
			healthAggregator,
		)
		dnsLogReporter.AddAggregator(
			NewDNSLogAggregator().
				AggregateOver(DNSAggregationKind(configParams.DNSLogsFileAggregationKind)).
				IncludeLabels(configParams.DNSLogsFileIncludeLabels).
				PerNodeLimit(configParams.DNSLogsFilePerNodeLimit),
			[]string{DNSLogsFileDispatcherName},
		)
		statsCollector.SetDNSLogReporter(dnsLogReporter)
	}

	if configParams.L7LogsFileEnabled {
		// Create the reporter, aggregator and dispatcher for L7 logging.
		l7LogReporter := NewL7LogReporter(
			map[string]LogDispatcher{
				L7LogsFileDispatcherName: NewFileDispatcher(
					configParams.L7LogsFileDirectory,
					L7LogFilename,
					configParams.L7LogsFileMaxFileSizeMB,
					configParams.L7LogsFileMaxFiles,
				),
			},
			configParams.L7LogsFlushInterval,
			healthAggregator,
		)
		// Create the aggregation kind
		aggKind := getL7AggregationKindFromConfigParams(configParams)
		l7LogReporter.AddAggregator(
			NewL7LogAggregator().
				AggregateOver(aggKind).
				PerNodeLimit(configParams.L7LogsFilePerNodeLimit),
			[]string{L7LogsFileDispatcherName},
		)
		statsCollector.SetL7LogReporter(l7LogReporter)
	}

	return statsCollector
}

// configureFlowAggregation adds appropriate aggregators to the FlowLogsReporter, depending on configuration.
func configureFlowAggregation(configParams *config.Config, cw *FlowLogsReporter) {
	addedFileAllow := false
	addedFileDeny := false
	if configParams.CloudWatchLogsReporterEnabled {
		if configParams.CloudWatchLogsEnabledForAllowed {
			log.Info("Creating Flow Logs Aggregator for allowed")
			caa := NewFlowLogAggregator().
				AggregateOver(FlowAggregationKind(configParams.CloudWatchLogsAggregationKindForAllowed)).
				IncludeLabels(configParams.CloudWatchLogsIncludeLabels).
				IncludePolicies(configParams.CloudWatchLogsIncludePolicies).
				MaxOriginalIPsSize(configParams.FlowLogsMaxOriginalIPsIncluded).
				PerFlowProcessLimit(configParams.FlowLogsFilePerFlowProcessLimit).
				ForAction(rules.RuleActionAllow)

			// Can we use the same aggregator for file logging?
			if configParams.FlowLogsFileEnabled &&
				configParams.FlowLogsFileEnabledForAllowed &&
				configParams.FlowLogsFileAggregationKindForAllowed == configParams.CloudWatchLogsAggregationKindForAllowed &&
				configParams.FlowLogsFileIncludeLabels == configParams.CloudWatchLogsIncludeLabels &&
				configParams.FlowLogsFileIncludePolicies == configParams.CloudWatchLogsIncludePolicies &&
				!configParams.FlowLogsFileIncludeService {
				log.Info("Adding Flow Logs Aggregator (allowed) for CloudWatch and File logs")
				cw.AddAggregator(caa, []string{CloudWatchLogsDispatcherName, FlowLogsFileDispatcherName})
				addedFileAllow = true
			} else {
				log.Info("Adding Flow Logs Aggregator (allowed) for CloudWatch logs")
				cw.AddAggregator(caa, []string{CloudWatchLogsDispatcherName})
			}
		}
		if configParams.CloudWatchLogsEnabledForDenied {
			log.Info("Creating Flow Logs Aggregator for denied")
			cad := NewFlowLogAggregator().
				AggregateOver(FlowAggregationKind(configParams.CloudWatchLogsAggregationKindForDenied)).
				IncludeLabels(configParams.CloudWatchLogsIncludeLabels).
				IncludePolicies(configParams.CloudWatchLogsIncludePolicies).
				MaxOriginalIPsSize(configParams.FlowLogsMaxOriginalIPsIncluded).
				PerFlowProcessLimit(configParams.FlowLogsFilePerFlowProcessLimit).
				ForAction(rules.RuleActionDeny)
			// Can we use the same aggregator for file logging?
			if configParams.FlowLogsFileEnabled &&
				configParams.FlowLogsFileEnabledForDenied &&
				configParams.FlowLogsFileAggregationKindForDenied == configParams.CloudWatchLogsAggregationKindForDenied &&
				configParams.FlowLogsFileIncludeLabels == configParams.CloudWatchLogsIncludeLabels &&
				configParams.FlowLogsFileIncludePolicies == configParams.CloudWatchLogsIncludePolicies &&
				!configParams.FlowLogsFileIncludeService {
				log.Info("Adding Flow Logs Aggregator (denied) for CloudWatch and File logs")
				cw.AddAggregator(cad, []string{CloudWatchLogsDispatcherName, FlowLogsFileDispatcherName})
				addedFileDeny = true
			} else {
				log.Info("Adding Flow Logs Aggregator (denied) for CloudWatch logs")
				cw.AddAggregator(cad, []string{CloudWatchLogsDispatcherName})
			}
		}
	}

	if configParams.FlowLogsFileEnabled {
		if !addedFileAllow && configParams.FlowLogsFileEnabledForAllowed {
			log.Info("Creating Flow Logs Aggregator for allowed")
			caa := NewFlowLogAggregator().
				AggregateOver(FlowAggregationKind(configParams.FlowLogsFileAggregationKindForAllowed)).
				IncludeLabels(configParams.FlowLogsFileIncludeLabels).
				IncludePolicies(configParams.FlowLogsFileIncludePolicies).
				IncludeService(configParams.FlowLogsFileIncludeService).
				IncludeProcess(configParams.FlowLogsCollectProcessInfo).
				IncludeTcpStats(configParams.FlowLogsCollectTcpStats).
				MaxOriginalIPsSize(configParams.FlowLogsMaxOriginalIPsIncluded).
				PerFlowProcessLimit(configParams.FlowLogsFilePerFlowProcessLimit).
				ForAction(rules.RuleActionAllow)
			log.Info("Adding Flow Logs Aggregator (allowed) for File logs")
			cw.AddAggregator(caa, []string{FlowLogsFileDispatcherName})
		}
		if !addedFileDeny && configParams.FlowLogsFileEnabledForDenied {
			log.Info("Creating Flow Logs Aggregator for denied")
			cad := NewFlowLogAggregator().
				AggregateOver(FlowAggregationKind(configParams.FlowLogsFileAggregationKindForDenied)).
				IncludeLabels(configParams.FlowLogsFileIncludeLabels).
				IncludePolicies(configParams.FlowLogsFileIncludePolicies).
				IncludeService(configParams.FlowLogsFileIncludeService).
				IncludeTcpStats(configParams.FlowLogsCollectTcpStats).
				IncludeProcess(configParams.FlowLogsCollectProcessInfo).
				MaxOriginalIPsSize(configParams.FlowLogsMaxOriginalIPsIncluded).
				PerFlowProcessLimit(configParams.FlowLogsFilePerFlowProcessLimit).
				ForAction(rules.RuleActionDeny)
			log.Info("Adding Flow Logs Aggregator (denied) for File logs")
			cw.AddAggregator(cad, []string{FlowLogsFileDispatcherName})
		}
	}
}
