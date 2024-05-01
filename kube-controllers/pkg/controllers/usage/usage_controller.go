package usage

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	uruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	crtlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/projectcalico/calico/kube-controllers/pkg/config"
	"github.com/projectcalico/calico/kube-controllers/pkg/controllers/controller"
	usagev1 "github.com/projectcalico/calico/libcalico-go/lib/apis/usage.tigera.io/v1"
	clientv3 "github.com/projectcalico/calico/libcalico-go/lib/clientv3"
)

// NewUsageController creates a controller that manages the creation of LicenseUsageReport objects in the datastore.
// This controller is responsible for creating a pipeline of components, connected by channels, that perform this task:
//
//	eventCollector: 	collects events that are of importance to report generation
//	      v
//	reportGenerator: 	generates basicLicenseUsageReports in response to events
//	      v
//	reportWriter: 		enriches basicLicenseUsageReports with additional context and writes them to the datastore as LicenseUsageReports
func NewUsageController(ctx context.Context, cfg *config.UsageControllerConfig, k8sClient *kubernetes.Clientset, calicoClient clientv3.Interface, informer cache.SharedIndexInformer) (controller.Controller, error) {
	usageClient, err := createUsageClient(cfg.RESTConfig)
	if err != nil {
		log.WithError(err).Error("Failed to create usage client")
		return nil, err
	}

	return &usageController{
		ctx:                ctx,
		k8sClient:          k8sClient,
		calicoClient:       calicoClient,
		usageClient:        usageClient,
		informer:           informer,
		usageReportsPerDay: cfg.UsageReportsPerDay,
	}, nil
}

func (c *usageController) Run(stopCh chan struct{}) {
	defer uruntime.HandleCrash()
	log.Info("Starting Usage Controller")

	// Establish the pipeline, with each component driving the next.
	c.collector = newEventCollector(stopCh, c.informer, c.usageReportsPerDay)
	c.reporter = newReportGenerator(c.collector.events, stopCh)
	c.writer = newReportWriter(c.reporter.reports, stopCh, c.ctx, c.k8sClient, c.calicoClient, c.usageClient)

	// Start the components.
	go c.collector.startCollectingEvents()
	go c.reporter.startGeneratingReports()
	go c.writer.startWriting()

	<-stopCh
	log.Info("Stopping Usage Controller")
}

type usageController struct {
	ctx                context.Context
	k8sClient          kubernetes.Interface
	calicoClient       clientv3.Interface
	usageClient        crtlclient.Client
	informer           cache.SharedIndexInformer
	usageReportsPerDay int

	collector eventCollector
	reporter  reportGenerator
	writer    reportWriter
}

// createUsageClient creates a client that can be used for working with usage.tigera.io/v1 GroupVersion objects.
func createUsageClient(cfg *rest.Config) (crtlclient.Client, error) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(usagev1.UsageGroupVersion, &usagev1.LicenseUsageReport{}, &usagev1.LicenseUsageReportList{})
	v1.AddToGroupVersion(scheme, usagev1.UsageGroupVersion)
	return crtlclient.New(cfg, crtlclient.Options{Scheme: scheme})
}

// mustSend sends the value on the channel, but will panic if the value is not received within 2 minutes. This should be
// used by goroutines that advance data along the usage controller pipeline to ensure that if the pipeline is severed,
// the program panics and does not block silently.
func mustSend[T any](channel chan T, value T) {
	t := time.NewTicker(time.Minute * 2)
	defer t.Stop()

	select {
	case channel <- value:
		return
	case <-t.C:
		panic(fmt.Sprintf("BUG: timed out sending to channel (%T) which should have a receiver ready", channel))
	}
}
