// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package dpi

import (
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"

	"github.com/projectcalico/kube-controllers/pkg/controllers/controller"
	"github.com/projectcalico/kube-controllers/pkg/controllers/worker"
)

type dpiController struct {
	r      *reconciler
	worker worker.Worker
}

func New(k8sClientset tigeraapi.Interface, dpiReconciler *reconciler) controller.Controller {
	w := worker.New(dpiReconciler)
	w.AddWatch(cache.NewListWatchFromClient(k8sClientset.ProjectcalicoV3().RESTClient(), "deeppacketinspections", "",
		fields.Everything()),
		&v3.DeepPacketInspection{})
	return &dpiController{
		r:      dpiReconciler,
		worker: w,
	}
}

func (c *dpiController) Run(stopCh chan struct{}) {
	log.Info("Starting DPI controller")
	go c.worker.Run(1, stopCh)
	<-stopCh
}
