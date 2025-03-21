package configfactory

import (
	"time"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewDefaultKubeControllersConfig(configName string) (*v3.KubeControllersConfiguration, error) {
	kubeControllersConfig := v3.NewKubeControllersConfiguration()
	kubeControllersConfig.Name = configName
	kubeControllersConfig.Spec = v3.KubeControllersConfigurationSpec{
		LogSeverityScreen:      "Info",
		HealthChecks:           v3.Enabled,
		EtcdV3CompactionPeriod: &v1.Duration{Duration: time.Minute * 10},
		Controllers: v3.ControllersConfig{
			Node: &v3.NodeControllerConfig{
				ReconcilerPeriod: &v1.Duration{Duration: time.Minute * 5},
				SyncLabels:       v3.Enabled,
				HostEndpoint: &v3.AutoHostEndpointConfig{
					AutoCreate:                v3.Disabled,
					CreateDefaultHostEndpoint: v3.DefaultHostEndpointsEnabled,
				},
				LeakGracePeriod: &v1.Duration{Duration: time.Minute * 15},
			},
			Policy: &v3.PolicyControllerConfig{
				ReconcilerPeriod: &v1.Duration{Duration: time.Minute * 5},
			},
			WorkloadEndpoint: &v3.WorkloadEndpointControllerConfig{
				ReconcilerPeriod: &v1.Duration{Duration: time.Minute * 5},
			},
			ServiceAccount: &v3.ServiceAccountControllerConfig{
				ReconcilerPeriod: &v1.Duration{Duration: time.Minute * 5},
			},
			Namespace: &v3.NamespaceControllerConfig{
				ReconcilerPeriod: &v1.Duration{Duration: time.Minute * 5},
			},
			LoadBalancer: &v3.LoadBalancerControllerConfig{
				AssignIPs: v3.AllServices,
			},
		},
	}
	return kubeControllersConfig, nil
}
