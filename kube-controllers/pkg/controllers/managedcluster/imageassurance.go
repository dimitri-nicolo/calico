// Copyright (c) 2022 Tigera, Inc. All rights reserved.

//go:build tesla
// +build tesla

package managedcluster

import (
	"github.com/projectcalico/calico/kube-controllers/pkg/config"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/controller"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/imageassuranceconfiguration"

	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"k8s.io/client-go/kubernetes"
)

type ImageAssurance struct {
	cfg config.GenericControllerConfig
}

func (i *ImageAssurance) CreateController(clusterName, ownerReference string, managedK8sCLI, managementK8sCLI kubernetes.Interface,
	managedCalicoCLI, managementCalicoCLI tigeraapi.Interface, restartChan chan<- string) controller.Controller {
	return imageassuranceconfiguration.New(clusterName, ownerReference, managedK8sCLI, managementK8sCLI, false, i.cfg, restartChan)
}

func (i *ImageAssurance) HandleManagedClusterRemoved(clusterName string) {
}

func (i *ImageAssurance) Initialize(stop chan struct{}, clusters ...string) {
}

func NewImageAssuranceController(cfg config.GenericControllerConfig) ControllerManager {
	return &ImageAssurance{cfg: cfg}
}
