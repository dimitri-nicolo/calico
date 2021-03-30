// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package license

import (
	"fmt"

	"github.com/projectcalico/kube-controllers/pkg/config"
	"github.com/projectcalico/kube-controllers/pkg/controllers/controller"
	"github.com/projectcalico/kube-controllers/pkg/controllers/worker"
	"github.com/projectcalico/kube-controllers/pkg/resource"
	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	tigeraapi "github.com/tigera/api/pkg/client/clientset_generated/clientset"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

type licenseController struct {
	clusterName string
	r           *reconciler
	worker      worker.Worker
	cfg         config.LicenseControllerCfg
}

func New(
	clusterName string,
	managedCalicoCLI tigeraapi.Interface,
	managementCalicoCLI tigeraapi.Interface,
	cfg config.LicenseControllerCfg) controller.Controller {
	r := NewLicenseReconciler(managedCalicoCLI, managementCalicoCLI, clusterName)

	// The high requeue attempts is because it's unlikely we would receive an event after failure to re trigger a
	// reconcile, meaning a temporary service disruption could lead to LicenseKey not being propagated.
	w := worker.New(r, worker.WithMaxRequeueAttempts(20))

	// monitor changes for the license in the managed cluster
	w.AddWatch(
		cache.NewListWatchFromClient(managedCalicoCLI.ProjectcalicoV3().RESTClient(), "licensekeys", "",
			fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.LicenseName))),
		&v3.LicenseKey{},
	)

	// monitor changes for the license that needs to be propagated
	w.AddWatch(
		cache.NewListWatchFromClient(managementCalicoCLI.ProjectcalicoV3().RESTClient(), "licensekeys", "",
			fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.LicenseName))),
		&v3.LicenseKey{},
	)

	return &licenseController{
		clusterName: clusterName,
		r:           r,
		worker:      w,
		cfg:         cfg,
	}
}

func (c *licenseController) Run(stop chan struct{}) {
	log.WithField("cluster", c.clusterName).Info("Starting License configuration controller")

	go c.worker.Run(c.cfg.NumberOfWorkers, stop)

	<-stop
}
