package config

import (
	"fmt"
	"net/url"
	"time"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/libcalico-go/lib/logutils"
)

const (
	ReportNameEnv  = "TIGERA_COMPLIANCE_REPORT_NAME"
	ReportStartEnv = "TIGERA_COMPLIANCE_REPORT_START_TIME"
	ReportEndEnv   = "TIGERA_COMPLIANCE_REPORT_END_TIME"
)

// Config contain environment based configuration for all compliance components. Although not all configuration is
// required for all components, it is useful having everything defined in one location.
type Config struct {
	// LogLevel
	LogLevel string `envconfig:"LOG_LEVEL"`

	// Health checks common to all components.
	HealthEnabled bool          `envconfig:"HEALTH_ENABLED" default:"true"`
	HealthPort    int           `envconfig:"HEALTH_PORT" default:"9099"`
	HealthHost    string        `envconfig:"HEALTH_HOST" default:"localhost"`
	HealthTimeout time.Duration `envconfig:"HEALTH_TIMEOUT" default:"30s"`

	// Elastic parameters
	ElasticURI         string `envconfig:"ELASTIC_URI"`
	ElasticScheme      string `envconfig:"ELASTIC_SCHEME" default:"http"`
	ElasticHost        string `envconfig:"ELASTIC_HOST" default:"elasticsearch-tigera-elasticsearch.calico-monitoring.svc.cluster.local"`
	ElasticPort        int    `envconfig:"ELASTIC_PORT" default:"9200"`
	ElasticUser        string `envconfig:"ELASTIC_USER" default:"elastic"`
	ElasticPassword    string `envconfig:"ELASTIC_PASSWORD"`
	ElasticCA          string `envconfig:"ELASTIC_CA"`
	ElasticIndexSuffix string `envconfig:"ELASTIC_INDEX_SUFFIX" default:"cluster"`

	// Snapshotter specific data.
	SnapshotHour int `envconfig:"TIGERA_COMPLIANCE_SNAPSHOT_HOUR" default:"0"`

	// Controller specific data.
	Namespace                  string        `envconfig:"TIGERA_COMPLIANCE_JOB_NAMESPACE" default:"calico-monitoring"`
	JobStartDelay              time.Duration `envconfig:"TIGERA_COMPLIANCE_JOB_START_DELAY" default:"30m"`
	MaxActiveJobs              int           `envconfig:"TIGERA_COMPLIANCE_MAX_ACTIVE_JOBS" default:"5"`
	MaxSuccessfulJobsHistory   int           `envconfig:"TIGERA_COMPLIANCE_MAX_SUCCESSFUL_JOBS_HISTORY" default:"2"`
	MaxFailedJobsHistory       int           `envconfig:"TIGERA_COMPLIANCE_MAX_FAILED_JOBS_HISTORY" default:"10"`
	IgnoreUnstartedReportAfter time.Duration `envconfig:"TIGERA_COMPLIANCE_IGNORE_UNSTARTED_REPORT_AFTER" default:"168h"`
	MaxJobRetries              int32         `envconfig:"TIGERA_COMPLIANCE_MAX_JOB_RETRIES" default:"10"`
	JobPollInterval            time.Duration `envconfig:"TIGERA_COMPLIANCE_JOB_POLL_INTERVAL" default:"10s"`
	JobNamePrefix              string        `envconfig:"TIGERA_COMPLIANCE_JOB_NAME_PREFIX" default:"compliance-reporter."`

	// Reporter specific data. Controller sets this through the environment names.
	ReportName  string `envconfig:"TIGERA_COMPLIANCE_REPORT_NAME"`
	ReportStart string `envconfig:"TIGERA_COMPLIANCE_REPORT_START_TIME"`
	ReportEnd   string `envconfig:"TIGERA_COMPLIANCE_REPORT_END_TIME"`

	// Pod annotation and init container and container regexes used to determine if Envoy is enabled inside the
	// pod. Used by the reporter and passed-thru from the controller.
	PodIstioSidecarAnnotation  string `envconfig:"TIGERA_COMPLIANCE_POD_ISTIO_SIDECAR_ANNOTATION" default:"sidecar.istio.io/status"`
	PodIstioInitContainerRegex string `envconfig:"TIGERA_COMPLIANCE_POD_ISTIO_INIT_CONTAINER_REGEX" default:".*/istio/proxy_init:.*"`
	PodIstioContainerRegex     string `envconfig:"TIGERA_COMPLIANCE_POD_ISTIO_CONTAINER_REGEX" default:".*/istio/proxy.*"`

	// Parsed values.
	ParsedReportStart time.Time
	ParsedReportEnd   time.Time
	ParsedElasticURL  *url.URL
	ParsedLogLevel    log.Level
}

func MustLoadConfig() *Config {
	c, err := LoadConfig()
	if err != nil {
		panic(err)
	}
	return c
}

func LoadConfig() (*Config, error) {
	var err error
	config := &Config{}
	err = envconfig.Process("", config)
	if err != nil {
		return nil, err
	}

	// Default the start/end times to now.
	now := time.Now()
	config.ParsedReportStart = now
	config.ParsedReportEnd = now

	// If the start/end times are specified, parse them now.
	if config.ReportStart != "" {
		config.ParsedReportStart, err = time.Parse(time.RFC3339, config.ReportStart)
		if err != nil {
			return nil, fmt.Errorf("report start time specified in environment is not RFC3339 formatted: %s", config.ReportStart)
		}
	}

	if config.ReportEnd != "" {
		config.ParsedReportEnd, err = time.Parse(time.RFC3339, config.ReportEnd)
		if err != nil {
			return nil, fmt.Errorf("report end time specified in environment is not RFC3339 formatted: %s", config.ReportEnd)
		}
	}

	if config.ParsedReportEnd.Before(config.ParsedReportStart) {
		return nil, fmt.Errorf(
			"invalid report times, end time is before start time: %s < %s",
			config.ParsedReportEnd.Format(time.RFC3339), config.ParsedReportStart.Format(time.RFC3339),
		)
	}

	// Parse elastic parms
	if config.ElasticURI != "" {
		config.ParsedElasticURL, err = url.Parse(config.ElasticURI)
		if err != nil {
			return nil, err
		}
	} else {
		config.ParsedElasticURL = &url.URL{
			Scheme: config.ElasticScheme,
			Host:   fmt.Sprintf("%s:%d", config.ElasticHost, config.ElasticPort),
		}
	}
	log.WithField("url", config.ParsedElasticURL).Debug("Parsed elastic url")

	// Parse log level.
	config.ParsedLogLevel = logutils.SafeParseLogLevel(config.LogLevel)

	// Check snapshot hour is within range.
	if config.SnapshotHour < 0 || config.SnapshotHour > 23 {
		return nil, fmt.Errorf(
			"invalid snapshot hour, should be in range 0-23: value=%d",
			config.SnapshotHour,
		)
	}

	return config, nil
}

func (c *Config) InitializeLogging() {
	log.SetFormatter(&logutils.Formatter{})
	log.AddHook(&logutils.ContextHook{})
	log.SetLevel(c.ParsedLogLevel)
}
