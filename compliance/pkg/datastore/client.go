// Copyright (c) 2020 Tigera, Inc. All rights reserved.
package datastore

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	calicoclient "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	clientv3 "github.com/tigera/api/pkg/client/clientset_generated/clientset/typed/projectcalico/v3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/projectcalico/calico/compliance/pkg/list"
)

type k8sInterface kubernetes.Interface
type calicoInterface clientv3.ProjectcalicoV3Interface

// ClientSet is a combined Calico/Kubernetes client set interface, with additional interfaces used by the compliance
// code.
type ClientSet interface {
	k8sInterface

	calicoInterface
	list.Source
}

func NewClientSet(k8sCli k8sInterface, calicoCli calicoInterface) ClientSet {
	return &clientSet{
		k8sInterface:    k8sCli,
		calicoInterface: calicoCli,
	}
}

// MustGetKubernetesClient returns a kubernetes client.
func MustGetKubernetesClient() kubernetes.Interface {
	config := MustGetConfig()

	// Build k8s client
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.WithError(err).Panic("Failed to load k8s client")
	}
	return k8sClient
}

func MustGetConfig() *rest.Config {
	kubeconfig := os.Getenv("KUBECONFIG")
	var config *rest.Config
	var err error
	if kubeconfig == "" {
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			log.WithError(err).Panic("Error getting in-cluster config")
		}
	} else {
		// creates a config from supplied kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			log.WithError(err).Panic("Error processing kubeconfig file in environment variable KUBECONFIG")
		}
	}
	config.Timeout = 15 * time.Second
	return config
}

// MustGetCalicoClient returns a Calico client.
func MustGetCalicoClient() clientv3.ProjectcalicoV3Interface {
	config := MustGetConfig()

	// Build calico client
	calicoClient, err := calicoclient.NewForConfig(config)
	if err != nil {
		log.Panicf("Failed to load Calico client: %v", err)
	}

	return calicoClient.ProjectcalicoV3()
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
