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
	if len(configPath) == 0 {
		return inCluster()
	}

	return outOfCluster(configPath)
}

func inCluster() (k8s *kubernetes.Clientset) {
	//creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Could not configure k8s client %s", err)
	}

	//creates the client for k8s
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Could not configure k8s client %s", err)
	}
	return client
}

func outOfCluster(configPath string) (k8s *kubernetes.Clientset) {
	//creates the ou-of-cluster config
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		log.Fatalf("Could not configure k8s client %s", err)
	}

	//creates the client for k8s
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Could not configure k8s client %s", err)
	}

	return client
}
