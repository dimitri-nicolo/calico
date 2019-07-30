package bootstrap

import (
	"log"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ConfigureK8sClient configures K8s client based on the configuration path. If no configuration is provided
// it will default to run as a pod. It will return nil if authNOn flag is set to false
func ConfigureK8sClient(configPath string) (k8s *kubernetes.Clientset) {
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

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to configure k8s client %s", err)
	}
	return client
}
