package policycalc

import (
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	"github.com/tigera/compliance/pkg/resources"
)

// ------
// This file contains all of the struct definitions that are used as input when instantiating a new policy calculator.
// ------

// The tiers containing the ordered set of Calico v3 resource types.
type Tier []resources.Resource
type Tiers []Tier

// The consistent set of configuration used for calculating policy impact.
type ResourceData struct {
	Tiers           Tiers
	Namespaces      []*corev1.Namespace
	ServiceAccounts []*corev1.ServiceAccount
}

// ModifiedResources is essentially a set of resource IDs used to track which resources were modified in the proposed
// update.
type ModifiedResources map[v3.ResourceID]bool

// Add adds a resource to the set of modified resources.
func (m ModifiedResources) Add(r resources.Resource) {
	m[resources.GetResourceID(r)] = true
}

// IsModified returns true if the specified resource is one of the resources that was modified in the proposed update.
func (m ModifiedResources) IsModified(r resources.Resource) bool {
	return m[resources.GetResourceID(r)]
}

// Config contain environment based configuration for the policy calculator.
type Config struct {
	// Whether or not the original action should be calculated. If this is false, the Action in the flow data will
	// be unchanged from the original value. If this is true, the Action will be calculated by passing the flow through
	// the current set of policy resources.
	CalculateOriginalAction bool `envconfig:"TIGERA_PIP_CALCULATE_ORIGINAL_ACTION"`
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
