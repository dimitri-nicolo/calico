package datastore

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	"github.com/projectcalico/libcalico-go/lib/clientv3"

	"github.com/tigera/compliance/pkg/list"
)

type k8sInterface kubernetes.Interface
type calicoInterface clientv3.Interface

// ClientSet is a combined Calico/Kubernetes client set interface, with additional interfaces used by the compliance
// code.
type ClientSet interface {
	k8sInterface
	calicoInterface
	list.Source
}

// MustGetKubernetesClient returns a kubernetes client.
func MustGetKubernetesClient() kubernetes.Interface {
	// Build kubeconfig path
	kubeconfig := os.Getenv("KUBECONFIG")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.WithError(err).Debug("failed to build config")
		panic(err)
	}
	config.Timeout = 15 * time.Second

	// Build k8s client
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.WithError(err).Debug("Failed to load k8s client")
		panic(err)
	}
	return k8sClient
}

// MustGetCalicoClient returns a Calico client.
func MustGetCalicoClient() clientv3.Interface {
	// Build calico client
	cfg, err := apiconfig.LoadClientConfig("")
	if err != nil {
		log.WithError(err).Error("failed to load datastore config")
		panic(err)
	}
	calicoClient, err := clientv3.New(*cfg)
	if err != nil {
		log.WithError(err).Error("Failed to load calico client")
		panic(err)
	}

	return calicoClient
}

// MustGetClientSet returns a client set (k8s and Calico)
func MustGetClientSet() ClientSet {
	return &clientSet{
		MustGetKubernetesClient(),
		MustGetCalicoClient(),
	}
}

type clientSet struct {
	k8sInterface
	calicoInterface
}
