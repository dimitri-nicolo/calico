package bootstrap

import (
	"log"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	clientv3 "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
)

// K8sClient is the actual client
type k8sClient struct {
	kubernetes.Interface
	clientv3.ProjectcalicoV3Interface
}

type K8sClient interface {
	kubernetes.Interface
	clientv3.ProjectcalicoV3Interface
}

// NewK8sClientWithConfig configures K8s client based a rest.Config.
func NewK8sClientWithConfig(cfg *rest.Config) K8sClient {
	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to configure k8s client %s", err)
	}

	calicoClient, err := calicoclient.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to configure calico client %s", err)
	}

	return &k8sClient{
		Interface:                k8s,
		ProjectcalicoV3Interface: calicoClient.ProjectcalicoV3(),
	}
}

// NewRestConfig creates a rest.Config. If this runs in k8s, it uses the credentials at fixed locations, otherwise, it
// uses flags.
func NewRestConfig(configPath string) *rest.Config {
	var (
		config *rest.Config
		err    error
	)

	if len(configPath) == 0 {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", configPath)
	}

	if err != nil {
		log.Fatalf("Failed to get k8s config %s", err)
	}

	return config
}
