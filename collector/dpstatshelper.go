// +build !windows

// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.

package collector

import (
	"strings"
	"time"

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

	//TODO: Move these into felix config
	DefaultAgeTimeout               = time.Duration(10) * time.Second
	DefaultInitialReportingDelay    = time.Duration(5) * time.Second
	DefaultExportingInterval        = time.Duration(1) * time.Second
	DefaultConntrackPollingInterval = time.Duration(5) * time.Second
)

// StartDataplaneStatsCollector creates the required dataplane stats collector, reporters and aggregators and starts
// collecting and reporting stats. Returns a collector that statistics should be reported to.
func StartDataplaneStatsCollector(configParams *config.Config, lookupsCache *calc.LookupsCache, healthAggregator *health.HealthAggregator) Collector {
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
			"directory": configParams.FlowLogsFileDirectory,
			"max_size":  configParams.FlowLogsFileMaxFileSizeMB,
			"max_files": configParams.FlowLogsFileMaxFiles,
		}).Info("Creating Flow Logs FileDispatcher")
		fd := NewFileDispatcher(
			configParams.FlowLogsFileDirectory,
			FlowLogFilename,
			configParams.FlowLogsFileMaxFileSizeMB,
			configParams.FlowLogsFileMaxFiles,
		)
		dispatchers[FlowLogsFileDispatcherName] = fd
	}
	if len(dispatchers) > 0 {
		log.Info("Creating Flow Logs Reporter")
		cw := NewFlowLogsReporter(dispatchers, configParams.FlowLogsFlushInterval, healthAggregator, configParams.FlowLogsEnableHostEndpoint)
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
			StatsDumpFilePath:            configParams.StatsDumpFilePath,
			NfNetlinkBufSize:             configParams.NfNetlinkBufSize,
			IngressGroup:                 1,
			EgressGroup:                  2,
			AgeTimeout:                   DefaultAgeTimeout,
			InitialReportingDelay:        DefaultInitialReportingDelay,
			ExportingInterval:            DefaultExportingInterval,
			ConntrackPollingInterval:     DefaultConntrackPollingInterval,
			EnableNetworkSets:            configParams.FlowLogsEnableNetworkSets,
			MaxOriginalSourceIPsIncluded: configParams.FlowLogsMaxOriginalIPsIncluded,
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
				IncludeLabels(configParams.DNSLogsFileIncludeLabels),
			[]string{DNSLogsFileDispatcherName},
		)
		statsCollector.SetDNSLogReporter(dnsLogReporter)
	}

	statsCollector.Start()
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
				ForAction(rules.RuleActionAllow)

			// Can we use the same aggregator for file logging?
			if configParams.FlowLogsFileEnabled &&
				configParams.FlowLogsFileEnabledForAllowed &&
				configParams.FlowLogsFileAggregationKindForAllowed == configParams.CloudWatchLogsAggregationKindForAllowed &&
				configParams.FlowLogsFileIncludeLabels == configParams.CloudWatchLogsIncludeLabels &&
				configParams.FlowLogsFileIncludePolicies == configParams.CloudWatchLogsIncludePolicies {
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
				ForAction(rules.RuleActionDeny)
			// Can we use the same aggregator for file logging?
			if configParams.FlowLogsFileEnabled &&
				configParams.FlowLogsFileEnabledForDenied &&
				configParams.FlowLogsFileAggregationKindForDenied == configParams.CloudWatchLogsAggregationKindForDenied &&
				configParams.FlowLogsFileIncludeLabels == configParams.CloudWatchLogsIncludeLabels &&
				configParams.FlowLogsFileIncludePolicies == configParams.CloudWatchLogsIncludePolicies {
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
				MaxOriginalIPsSize(configParams.FlowLogsMaxOriginalIPsIncluded).
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
				MaxOriginalIPsSize(configParams.FlowLogsMaxOriginalIPsIncluded).
				ForAction(rules.RuleActionDeny)
			log.Info("Adding Flow Logs Aggregator (denied) for File logs")
			cw.AddAggregator(cad, []string{FlowLogsFileDispatcherName})
		}
	}
}
