// Copyright (c) 2022 Tigera Inc. All rights reserved.

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	v1 "k8s.io/api/core/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	bapi "github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	lincensing_client "github.com/projectcalico/calico/licensing/client"

	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"

	"github.com/projectcalico/calico/licensing/client/features"
	"github.com/projectcalico/calico/licensing/monitor"
	"github.com/projectcalico/calico/lma/pkg/elastic"

	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/cache"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/config"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/managedcluster"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/namespace"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/policyrecommendation"
	"github.com/projectcalico/calico/continuous-policy-recommendation/pkg/stagednetworkpolicies"

	log "github.com/sirupsen/logrus"
)

// backendClientAccessor is an interface to access the backend client from the main v2 client.
type backendClientAccessor interface {
	Backend() bapi.Client
}

func main() {
	var err error
	policyrecommendationConfig, err := config.LoadConfig()
	if err != nil {
		panic(err.Error())
	}

	policyrecommendationConfig.InitializeLogging()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientFactory := lmak8s.NewClientSetFactory(
		policyrecommendationConfig.MultiClusterForwardingCA,
		policyrecommendationConfig.MultiClusterForwardingEndpoint,
	)

	clientSet, err := clientFactory.NewClientSetForApplication(lmak8s.DefaultCluster)
	if err != nil {
		panic(err.Error())
	}

	envCfg := elastic.MustLoadConfig()
	esClientFactory := elastic.NewClusterContextClientFactory(envCfg)
	lmaESClient, err := esClientFactory.ClientForCluster(lmak8s.DefaultCluster)

	if err != nil {
		log.WithError(err).Fatal("Could not connect to Elasticsearch")
	}
	if err := lmaESClient.CreateEventsIndex(ctx); err != nil {
		log.WithError(err).Fatal("Failed to create events index")
	}

	// setup license check
	v3Client, err := clientv3.NewFromEnv()
	if err != nil {
		log.WithError(err).Fatal("Failed to build v3 Calico client")
	}

	// Define some of the callbacks for the license monitor. Any changes
	// just send a signal back on the license changed channel.
	licenseMonitor := monitor.New(v3Client.(backendClientAccessor).Backend())
	err = licenseMonitor.RefreshLicense(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to get license from datastore; continuing without a license")
	}

	licenseUpdateChan := make(chan struct{})

	licenseMonitor.SetFeaturesChangedCallback(
		func() {
			licenseUpdateChan <- struct{}{}
		},
	)

	licenseMonitor.SetStatusChangedCallback(
		func(newLicenseStatus lincensing_client.LicenseStatus) {
			licenseUpdateChan <- struct{}{}
		},
	)

	// Start the license monitor, which will trigger the callback above at start of day and then whenever the license
	// status changes.
	go func() {
		err := licenseMonitor.MonitorForever(context.Background())
		if err != nil {
			log.WithError(err).Warn("Error while continuously monitoring the license.")
		}
	}()

	// setup caches

	// SNP cache
	snpResourceCache := cache.NewSynchronizedObjectCache[*v3.StagedNetworkPolicy]()

	// Namespace cache
	namespaceCache := cache.NewSynchronizedObjectCache[*v1.Namespace]()

	// create main controller
	managementStandalonePolicyRecController :=
		policyrecommendation.NewPolicyRecommendationController(
			clientSet.ProjectcalicoV3(),
			lmaESClient,
		)
	managedclusterController := managedcluster.NewManagedClusterController(clientSet.ProjectcalicoV3(), clientFactory, esClientFactory)
	stagednetworkpolicyController := stagednetworkpolicies.NewStagedNetworkPolicyController(clientSet.ProjectcalicoV3(), snpResourceCache)
	namespaceController := namespace.NewNamespaceController(clientSet, namespaceCache)

	// setup shutdown sigs
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	hasLicense := licenseMonitor.GetFeatureStatus(features.PolicyRecommendation)
	controllerRunning := false

	for {
		if hasLicense && !controllerRunning {
			managementStandalonePolicyRecController.Run(ctx)
			managedclusterController.Run(ctx)
			stagednetworkpolicyController.Run(ctx)
			namespaceController.Run(ctx)
			controllerRunning = true
		} else if !hasLicense && controllerRunning {
			managementStandalonePolicyRecController.Close()
			managedclusterController.Close()
			stagednetworkpolicyController.Close()
			namespaceController.Close()
			controllerRunning = false
		}

		select {
		case <-licenseUpdateChan:
			log.Info("License status has changed")
			hasLicense = licenseMonitor.GetFeatureStatus(features.PolicyRecommendation)
			continue
		case <-shutdown:
			log.Info("exiting")
			if controllerRunning {
				managementStandalonePolicyRecController.Close()
				managedclusterController.Close()
			}
			return
		}
	}
}
