package bootstrap

import (
	"log"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	calicoclient "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset"
	clientv3 "github.com/tigera/calico-k8sapiserver/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
)

// K8sClient is the actual client
type K8sClient struct {
	kubernetes.Interface
	clientv3.ProjectcalicoV3Interface
}

// ConfigureK8sClient configures K8s client based on the configuration path. If no configuration is provided
// it will default to run as a pod. It will return nil if authNOn flag is set to false
func ConfigureK8sClient(configPath string) (*K8sClient, *rest.Config) {
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

	k8s, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to configure k8s client %s", err)
	}

	calicoClient, err := calicoclient.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to configure calico client %s", err)
	}

	return &K8sClient{
		Interface:                k8s,
		ProjectcalicoV3Interface: calicoClient.ProjectcalicoV3(),
	}, config
}
