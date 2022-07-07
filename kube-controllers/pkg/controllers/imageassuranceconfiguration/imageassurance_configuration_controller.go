// Copyright (c) 2022 Tigera, Inc. All rights reserved.

//go:build tesla
// +build tesla

package imageassuranceconfiguration

import (
	"fmt"

	"github.com/projectcalico/calico/kube-controllers/pkg/config"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/controller"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/utils"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/worker"
	"github.com/projectcalico/calico/kube-controllers/pkg/resource"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type imageAssuranceConfigController struct {
	clusterName string
	r           *reconciler
	worker      worker.Worker
	cfg         config.ImageAssuranceConfig
}

func New(
	clusterName string,
	ownerReference string,
	managedK8sCLI kubernetes.Interface,
	managementK8sCLI kubernetes.Interface,
	management bool,
	cfg config.ImageAssuranceConfig,
	restartChan chan<- string,
) controller.Controller {
	logCtx := log.WithField("cluster", clusterName)

	r := &reconciler{
		clusterName:                        clusterName,
		ownerReference:                     ownerReference,
		management:                         management,
		managementK8sCLI:                   managementK8sCLI,
		managedK8sCLI:                      managedK8sCLI,
		restartChan:                        restartChan,
		imageAssuranceNamespace:            resource.ImageAssuranceNameSpaceName,
		admissionControllerClusterRoleName: cfg.AdmissionControllerClusterRoleName,
		intrusionDetectionClusterRoleName:  cfg.IntrusionDetectionControllerClusterRoleName,
		scannerClusterRoleName:             cfg.ScannerClusterRoleName,
		podWatcherClusterRoleName:          cfg.PodWatcherClusterRoleName,
	}

	// The high requeue attempts is because it's unlikely we would receive an event after failure to re trigger a
	// reconcile, meaning a temporary service disruption could lead to image assurance credentials not being propagated.
	w := worker.New(r, worker.WithMaxRequeueAttempts(20))

	var err error
	utils.AddWatchForActiveOperator(w, r.managementK8sCLI)
	// We need to get the operator namespace because we need to watch secrets in that namespace.
	// If we are unable to successfully read the namespace assume the default operator namespace.
	// We also setup a watch for the ConfigMap with the namespace so if our assumption is wrong we
	// will be triggered when it is available or updated and a Reconcile will trigger a restart so
	// this controller can be restarted and pick up the correct namespace.
	r.managementOperatorNamespace, err = utils.FetchOperatorNamespace(r.managementK8sCLI)
	if err != nil {
		r.managementOperatorNamespace = utils.DefaultTigeraOperatorNamespace
		logCtx.WithField("cluster", "management").WithField("message", err.Error()).
			Info("unable to fetch operator namespace, assuming active operator namespace is tigera-operator")
	}

	// The items in this block are only for a managed cluster because they either already exists in the appropriate
	// namespace in the management cluster or it is for the admission controller which is not currently used in
	// the admission controller.
	if !r.management {
		// The managed cluster might not be set up for image assurance so we watch the namespace so we're notified
		// when it becomes available.
		w.AddWatch(
			cache.NewListWatchFromClient(managedK8sCLI.CoreV1().RESTClient(), "namespaces", "",
				fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", r.imageAssuranceNamespace))),
			&corev1.Namespace{}, worker.ResourceWatchAdd, worker.ResourceWatchUpdate, worker.ResourceWatchDelete,
		)

		// In managed cluster we need to watch tigera-image-assurance-api-cert (which contains CA cert for image
		// assurance api) for updates and deletes.
		w.AddWatch(
			cache.NewListWatchFromClient(managedK8sCLI.CoreV1().RESTClient(), "secrets", r.imageAssuranceNamespace,
				fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.ImageAssuranceAPICertSecretName))),
			&corev1.Secret{}, worker.ResourceWatchUpdate, worker.ResourceWatchDelete,
		)

		// In managed cluster we need to watch tigera-image-assurance-api-pod-blocker-api-access key, which contains
		// service account token for accessing image assurance api for updates and deletes.
		w.AddWatch(
			cache.NewListWatchFromClient(managedK8sCLI.CoreV1().RESTClient(), "secrets", r.imageAssuranceNamespace,
				fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.ManagedIAAdmissionControllerResourceName))),
			&corev1.Secret{}, worker.ResourceWatchUpdate, worker.ResourceWatchDelete,
		)

		w.AddWatch(
			cache.NewListWatchFromClient(managedK8sCLI.CoreV1().RESTClient(), "configmaps", r.imageAssuranceNamespace,
				fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.ImageAssuranceConfigMapName))),
			&corev1.ConfigMap{},
			worker.ResourceWatchUpdate, worker.ResourceWatchDelete,
		)

		w.AddWatch(
			cache.NewListWatchFromClient(managementK8sCLI.CoreV1().RESTClient(), "secrets", r.managementOperatorNamespace,
				fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.ImageAssuranceAPICertPairSecretName))),
			&corev1.Secret{},
			worker.ResourceWatchUpdate, worker.ResourceWatchDelete, worker.ResourceWatchAdd,
		)

		w.AddWatch(
			cache.NewListWatchFromClient(managementK8sCLI.CoreV1().RESTClient(), "configmaps", r.managementOperatorNamespace,
				fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.ImageAssuranceConfigMapName))),
			&corev1.ConfigMap{},
			worker.ResourceWatchUpdate, worker.ResourceWatchDelete, worker.ResourceWatchAdd,
		)

		// Watch for changes to the service accounts created by the reconciler.
		w.AddWatch(
			cache.NewListWatchFromClient(managementK8sCLI.CoreV1().RESTClient(), "serviceaccounts", r.managementOperatorNamespace,
				fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", fmt.Sprintf(resource.ManagementIAAdmissionControllerResourceNameFormat,
					r.clusterName)))),
			&corev1.ServiceAccount{},
			worker.ResourceWatchUpdate, worker.ResourceWatchDelete, worker.ResourceWatchAdd,
		)
	}

	if management {
		// Watch for changes to the service accounts created by the reconciler.
		w.AddWatch(
			cache.NewListWatchFromClient(managementK8sCLI.CoreV1().RESTClient(), "serviceaccounts", r.managementOperatorNamespace,
				fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.ImageAssuranceIDSControllerServiceAccountName))),
			&corev1.ServiceAccount{},
			worker.ResourceWatchUpdate, worker.ResourceWatchDelete, worker.ResourceWatchAdd,
		)

		// Watch for changes to the service accounts created by the reconciler.
		w.AddWatch(
			cache.NewListWatchFromClient(managementK8sCLI.CoreV1().RESTClient(), "serviceaccounts", r.managementOperatorNamespace,
				fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.ImageAssuranceScannerServiceAccountName))),
			&corev1.ServiceAccount{},
			worker.ResourceWatchUpdate, worker.ResourceWatchDelete, worker.ResourceWatchAdd,
		)

		// Watch for changes to the service accounts created by the reconciler.
		w.AddWatch(
			cache.NewListWatchFromClient(managementK8sCLI.CoreV1().RESTClient(), "serviceaccounts", r.managementOperatorNamespace,
				fields.ParseSelectorOrDie(fmt.Sprintf("metadata.name=%s", resource.ImageAssurancePodWatcherServiceAccountName))),
			&corev1.ServiceAccount{},
			worker.ResourceWatchUpdate, worker.ResourceWatchDelete, worker.ResourceWatchAdd,
		)
	}

	logCtx.Info("Watching for management cluster configuration changes.")

	return &imageAssuranceConfigController{
		clusterName: clusterName,
		r:           r,
		worker:      w,
		cfg:         cfg,
	}
}

func (c *imageAssuranceConfigController) Run(stop chan struct{}) {
	log.WithField("cluster", c.clusterName).Info("Starting image assurance configuration controller")

	go c.worker.Run(c.cfg.NumberOfWorkers, stop)

	<-stop
}
