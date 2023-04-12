// Copyright (c) 2023 Tigera, Inc. All rights reserved.
//

package templates

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/json"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
)

// Template is the internal representation of an Elastic template
type Template struct {
	IndexPatterns []string               `json:"index_patterns,omitempty"`
	Settings      map[string]interface{} `json:"settings,omitempty"`
	Mappings      map[string]interface{} `json:"mappings,omitempty"`
}

// TemplateConfig is the configuration used to create a template
// in Elastic. A template has associated an ILM policy, index patterns
// mappings, settings and a bootstrap index to perform rollover
type TemplateConfig struct {
	logsType    bapi.DataType
	info        bapi.ClusterInfo
	application string
	shards      int
	replicas    int
}

// NewTemplateConfig will build a TemplateConfig based on the logs type, cluster information
// and provided Option(s)
func NewTemplateConfig(logsType bapi.DataType, info bapi.ClusterInfo, opts ...Option) *TemplateConfig {
	defaultConfig := &TemplateConfig{logsType: logsType, info: info, shards: 1, replicas: 0, application: "fluentd"}

	for _, opt := range opts {
		defaultConfig = opt(defaultConfig)
	}

	return defaultConfig
}

// Option will customize different values for a TemplateConfig
type Option func(config *TemplateConfig) *TemplateConfig

// WithReplicas will set the number of replicas to be used
// for an index template
func WithReplicas(replicas int) Option {
	return func(config *TemplateConfig) *TemplateConfig {
		config.replicas = replicas
		return config
	}
}

// WithShards will set the number of shards to be used
// for an index template
func WithShards(shards int) Option {
	return func(config *TemplateConfig) *TemplateConfig {
		config.shards = shards
		return config
	}
}

// WithApplication will set the application name to be
// used when constructing a boostrap index or index alias
func WithApplication(application string) Option {
	return func(config *TemplateConfig) *TemplateConfig {
		config.application = application
		return config
	}
}

// TemplateName will provide the name of the template
func (c *TemplateConfig) TemplateName() string {
	if c.info.Tenant == "" {
		return fmt.Sprintf("tigera_secure_ee_%s.%s.", strings.ToLower(string(c.logsType)), c.info.Cluster)
	}

	return fmt.Sprintf("tigera_secure_ee_%s.%s.%s.", strings.ToLower(string(c.logsType)), c.info.Tenant, c.info.Cluster)
}

func (c *TemplateConfig) indexPatterns() string {
	prefix, ok := IndexPatternsPrefixLookup[c.logsType]
	if !ok {
		panic("index prefix for log type not implemented")
	}

	if c.info.Tenant == "" {
		return fmt.Sprintf("%s.%s.*", prefix, c.info.Cluster)
	}

	return fmt.Sprintf("%s.%s.%s.*", prefix, c.info.Tenant, c.info.Cluster)
}

func (c *TemplateConfig) mappings() string {
	switch c.logsType {
	case bapi.FlowLogs:
		return FlowLogMappings
	case bapi.DNSLogs:
		return DNSLogMappings
	case bapi.L7Logs:
		return L7LogMappings
	case bapi.AuditKubeLogs, bapi.AuditEELogs:
		return AuditMappings
	case bapi.BGPLogs:
		return BGPMappings
	case bapi.Events:
		return EventsMappings
	case bapi.WAFLogs:
		return WAFMappings
	case bapi.ReportData:
		return ReportMappings
	case bapi.Benchmarks:
		return BenchmarksMappings
	case bapi.Snapshots:
		return SnapshotMappings
	case bapi.RuntimeReports:
		return RuntimeReportsMappings
	default:
		panic("log type not implemented")
	}
}

// Alias will provide the alias used to write data
func (c *TemplateConfig) Alias() string {
	if c.info.Tenant == "" {
		return fmt.Sprintf("tigera_secure_ee_%s.%s.", c.logsType, c.info.Cluster)
	}
	return fmt.Sprintf("tigera_secure_ee_%s.%s.%s.", c.logsType, c.info.Tenant, c.info.Cluster)
}

func (c *TemplateConfig) ilmPolicyName() string {
	return fmt.Sprintf("tigera_secure_ee_%s_policy", c.logsType)
}

// BootstrapIndexName will construct the boostrap index name
// to be used for rollover
func (c *TemplateConfig) BootstrapIndexName() string {
	if c.info.Tenant == "" {
		return fmt.Sprintf("<tigera_secure_ee_%s.%s.%s-{now/s{yyyyMMdd}}-000001>",
			c.logsType, c.info.Cluster, c.application)
	}

	return fmt.Sprintf("<tigera_secure_ee_%s.%s.%s.%s-{now/s{yyyyMMdd}}-000001>",
		c.logsType, c.info.Tenant, c.info.Cluster, c.application)
}

func (c *TemplateConfig) settings() map[string]interface{} {
	// DNS logs requires additional settings to
	// number of shards and replicas
	indexSettings := c.initIndexSettings()
	indexSettings["number_of_shards"] = c.shards
	indexSettings["number_of_replicas"] = c.replicas

	lifeCycle := make(map[string]interface{})
	// ILM policy is created by the operator and only
	// referenced by the template
	lifeCycle["name"] = c.ilmPolicyName()
	lifeCycle["rollover_alias"] = c.Alias()
	indexSettings["lifecycle"] = lifeCycle

	return indexSettings
}

// initIndexSettings will unmarshal other indexSettings for the index
// (that do not cover number of shards and replicas) if they have been
// defined in SettingsLookup or an empty map otherwise
func (c *TemplateConfig) initIndexSettings() map[string]interface{} {
	settingsName, ok := SettingsLookup[c.logsType]
	if !ok {
		return make(map[string]interface{})
	}

	indexSettings, err := unmarshal(settingsName)
	if err != nil {
		logrus.WithError(err).Fatal("failed to parse dns settings from embedded file")
	}

	return indexSettings
}

// Build will create an internal representation of the
// template to be created in Elastic
func (c *TemplateConfig) Build() (*Template, error) {
	mappings := c.mappings()
	settings := c.settings()
	indexPatterns := c.indexPatterns()

	// Convert mapping to map[string]interface{}
	indexMappings, err := unmarshal(mappings)
	if err != nil {
		return nil, err
	}

	return &Template{
		IndexPatterns: []string{indexPatterns},
		Settings:      settings,
		Mappings:      indexMappings,
	}, nil
}

func unmarshal(source string) (map[string]interface{}, error) {
	var value map[string]interface{}
	if err := json.Unmarshal([]byte(source), &value); err != nil {
		return nil, err
	}
	return value, nil
}
