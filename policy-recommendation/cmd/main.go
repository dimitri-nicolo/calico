// Copyright (c) 2022-2023 Tigera Inc. All rights reserved.

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"k8s.io/apimachinery/pkg/runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/api/core/v1"

	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	bapi "github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	lincensing_client "github.com/projectcalico/calico/licensing/client"
	"github.com/projectcalico/calico/licensing/client/features"
	"github.com/projectcalico/calico/licensing/monitor"
	lsclient "github.com/projectcalico/calico/linseed/pkg/client"
	lsrest "github.com/projectcalico/calico/linseed/pkg/client/rest"
	lmak8s "github.com/projectcalico/calico/lma/pkg/k8s"
	"github.com/projectcalico/calico/policy-recommendation/pkg/cache"
	"github.com/projectcalico/calico/policy-recommendation/pkg/config"
	"github.com/projectcalico/calico/policy-recommendation/pkg/managedcluster"
	"github.com/projectcalico/calico/policy-recommendation/pkg/namespace"
	"github.com/projectcalico/calico/policy-recommendation/pkg/policyrecommendation"
	"github.com/projectcalico/calico/policy-recommendation/pkg/stagednetworkpolicies"
	"github.com/projectcalico/calico/policy-recommendation/pkg/syncer"
	"github.com/projectcalico/calico/policy-recommendation/utils"
)

// backendClientAccessor is an interface to access the backend client from the main v2 client.
type backendClientAccessor interface {
	Backend() bapi.Client
}

func main() {
	var err error
	policyrecommendationConfig, err := config.LoadConfig()
	if err != nil {
		log.WithError(err).Fatal("Failed to load  policy recommendation config")
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

	restConfig := clientFactory.NewRestConfigForApplication(lmak8s.DefaultCluster)
	scheme := runtime.NewScheme()
	if err = v3.AddToScheme(scheme); err != nil {
		log.WithError(err).Fatal("Failed to configure controller runtime client")
	}

	// client is used to get ManagedCluster resources in both single-tenant and multi-tenant modes.
	client, err := ctrlclient.NewWithWatch(restConfig, ctrlclient.Options{Scheme: scheme})
	if err != nil {
		log.WithError(err).Fatal("Failed to configure controller runtime client with watch")
	}

	// Create linseed Client.
	lsConfig := lsrest.Config{
		URL:             policyrecommendationConfig.LinseedURL,
		CACertPath:      policyrecommendationConfig.LinseedCA,
		ClientKeyPath:   policyrecommendationConfig.LinseedClientKey,
		ClientCertPath:  policyrecommendationConfig.LinseedClientCert,
		FIPSModeEnabled: policyrecommendationConfig.FIPSModeEnabled,
	}
	linseedClient, err := lsclient.NewClient(policyrecommendationConfig.TenantID, lsConfig, lsrest.WithTokenPath(policyrecommendationConfig.LinseedToken))
	if err != nil {
		log.WithError(err).Fatal("failed to create linseed client")
	}

	// setup license check
	v3Client, err := clientv3.NewFromEnv()
	if err != nil {
		log.WithError(err).Fatal("Failed to build v3 Calico client")
	}

	clusterDomain, err := utils.GetClusterDomain(utils.DefaultResolveConfPath)
	if err != nil {
		clusterDomain = utils.DefaultClusterDomain
		log.WithError(err).Errorf("Couldn't find the cluster domain from the resolv.conf, defaulting to %s", clusterDomain)
	} else {
		log.Debugf("clusterDomain: %s", clusterDomain)
	}
	serviceNameSuffix := utils.GetServiceNameSuffix(clusterDomain)

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

	// Start the license monitor, which will trigger the callback above at start of day and then
	// whenever the license status changes.
	go func() {
		err := licenseMonitor.MonitorForever(context.Background())
		if err != nil {
			log.WithError(err).Warn("Error while continuously monitoring the license.")
		}
	}()

	// Setup Caches
	// StagedNetworkPolicy cache
	snpResourceCache := cache.NewSynchronizedObjectCache[*v3.StagedNetworkPolicy]()

	// Namespace cache
	namespaceCache := cache.NewSynchronizedObjectCache[*v1.Namespace]()

	// NetworkSets Cache
	networkSetCache := cache.NewSynchronizedObjectCache[*v3.NetworkSet]()

	// Cache set
	caches := &syncer.CacheSet{
		Namespaces:            namespaceCache,
		NetworkSets:           networkSetCache,
		StagedNetworkPolicies: snpResourceCache,
	}

	// Setup Synchronizer
	cacheSynchronizer := syncer.NewCacheSynchronizer(clientSet, *caches, utils.SuffixGenerator)

	// Controller Setup
	// create main controller
	suffixGenerator := utils.SuffixGenerator
	managementStandalonePolicyRecController := policyrecommendation.NewPolicyRecommendationController(
		clientSet.ProjectcalicoV3(),
		linseedClient,
		cacheSynchronizer,
		caches,
		lmak8s.DefaultCluster,
		serviceNameSuffix,
		&suffixGenerator,
	)
	managedclusterController := managedcluster.NewManagedClusterController(ctx,
		client,
		clientFactory,
		linseedClient,
		policyrecommendationConfig.TenantNamespace,
	)
	stagednetworkpolicyController := stagednetworkpolicies.NewStagedNetworkPolicyController(
		clientSet.ProjectcalicoV3(),
		snpResourceCache,
	)
	namespaceController := namespace.NewNamespaceController(
		clientSet,
		namespaceCache,
		cacheSynchronizer,
	)

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
