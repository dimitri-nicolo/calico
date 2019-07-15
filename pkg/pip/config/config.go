package config

import (
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

// Config contain environment based configuration for the policy calculator.
type Config struct {
	// Whether or not the original action should be calculated. If this is false, the Action in the flow data will
	// be unchanged from the original value. If this is true, the Action in the flow log will be calculated by
	// passing the flow through the original set of policy resources.
	CalculateOriginalAction bool `envconfig:"TIGERA_PIP_CALCULATE_ORIGINAL_ACTION"`

	// Whether calico-managed endpoints always perform IP matches against NetworkSets and Nets rule matchers. This is
	// relevant for how flows are processed if they do not contain IP address information.
	// -  If set the false (the default), a Calico endpoint with no IP will be treated as a no-match against a
	//    NetworkSet or Nets rule match. The assumption is that label selection is used for Calico managed endpoints and
	//    NetworkSets and Nets rule matchers would only contain CIDRs outside the pod CIDR ranges.
	// -  If set to true, a Calico endpoint with no IP will have an indeterminate match against a NetworkSet or Nets
	//    rule match, this may in turn result in an unknown action for a flow.
	CalicoEndpointNetMatchAlways bool `envconfig:"TIGERA_PIP_CALICO_EP_NET_MATCH_ALWAYS"`

	// Whether flow log data should be augmented with audit log data. Audit log data can provide service account
	// information for pods, and named ports for pods and host endpoints. When enabled, snapshots and audit logs are
	// replayed to determine the cluster configuration for the same time period as the flow log query.
	AugmentFlowLogDataWithAuditLogData bool `envconfig:"TIGERA_PIP_AUGMENT_FLOW_WITH_AUDIT"`

	// Whether flow log data should be augmented with current configuration. Configuration queries directly from etcd
	// or k8s can be used to augment the flow log data. This can work in conjunction with the
	// `AugmentFlowLogDataWithAuditLogData` option, and this is always applied last.
	AugmentFlowLogDataWithCurrentConfiguration bool `envconfig:"TIGERA_PIP_AUGMENT_FLOW_WITH_CURRENT"`
}

// MustLoadConfig loads the configuration from the environment variables. It panics if there is an error in the config.
func MustLoadConfig() *Config {
	c, err := LoadConfig()
	if err != nil {
		log.Panicf("Error loading configuration: %v", err)
	}
	return c
}

// LoadConfig loads the configuration from the environment variables.
func LoadConfig() (*Config, error) {
	var err error
	config := &Config{}
	err = envconfig.Process("", config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
