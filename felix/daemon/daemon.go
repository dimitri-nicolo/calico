// Copyright (c) 2021 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package daemon

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	k8sresources "github.com/projectcalico/calico/libcalico-go/lib/backend/k8s/resources"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	lclient "github.com/projectcalico/calico/licensing/client"
	"github.com/projectcalico/calico/licensing/client/features"
	"github.com/projectcalico/calico/licensing/monitor"

	apiv3 "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
	libapiv3 "github.com/projectcalico/calico/libcalico-go/lib/apis/v3"
	"github.com/projectcalico/calico/libcalico-go/lib/backend"
	bapi "github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/k8s"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/syncersv1/felixsyncer"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/syncersv1/updateprocessors"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/watchersyncer"
	client "github.com/projectcalico/calico/libcalico-go/lib/clientv3"
	cerrors "github.com/projectcalico/calico/libcalico-go/lib/errors"
	"github.com/projectcalico/calico/libcalico-go/lib/health"
	lclogutils "github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/libcalico-go/lib/options"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
	"github.com/projectcalico/calico/pod2daemon/binder"
	"github.com/projectcalico/calico/typha/pkg/discovery"
	"github.com/projectcalico/calico/typha/pkg/syncclient"

	"github.com/projectcalico/calico/felix/buildinfo"
	"github.com/projectcalico/calico/felix/calc"
	"github.com/projectcalico/calico/felix/capture"
	"github.com/projectcalico/calico/felix/collector"
	"github.com/projectcalico/calico/felix/config"
	dp "github.com/projectcalico/calico/felix/dataplane"
	"github.com/projectcalico/calico/felix/jitter"
	"github.com/projectcalico/calico/felix/logutils"
	"github.com/projectcalico/calico/felix/policysync"
	"github.com/projectcalico/calico/felix/proto"
	"github.com/projectcalico/calico/felix/statusrep"
)

const (
	// Our default value for GOGC if it is not set.  This is the percentage that heap usage must
	// grow by to trigger a garbage collection.  Go's default is 100, meaning that 50% of the
	// heap can be lost to garbage.  We reduce it to this value to trade increased CPU usage for
	// lower occupancy.
	defaultGCPercent = 20

	// String sent on the failure report channel to indicate we're shutting down for config
	// change.
	reasonConfigChanged        = "config changed"
	reasonEncapChanged         = "encapsulation changed"
	reasonLicenseConfigChanged = "license config changed"
	// String sent on the failure report channel to indicate we're shutting down for a child
	// process exited. e.g. charon daemon.
	reasonChildExited = "child exit"
	reasonFatalError  = "fatal error"
	// Process return code used to report a config change.  This is the same as the code used
	// by SIGHUP, which means that the wrapper script also restarts Felix on a SIGHUP.
	configChangedRC = 129

	// Grace period we allow for graceful shutdown before panicking.
	gracefulShutdownTimeout = 30 * time.Second

	// Process return code used to report a child exit.  This is the same as the code used
	// by SIGHUP, which means that the wrapper script also restarts Felix on a SIGHUP.
	childExitedRC = 129

	// CloudWatch Health metrics names.
	healthyNodeMetricName   = "Nodes reporting healthy"
	unHealthyNodeMetricName = "Nodes reporting unhealthy"
)

// Run is the entry point to run a Felix instance.
//
// Its main role is to sequence Felix's startup by:
//
// Initialising early logging config (log format and early debug settings).
//
// Parsing command line parameters.
//
// Loading datastore configuration from the environment or config file.
//
// Loading more configuration from the datastore (this is retried until success).
//
// Starting the configured internal (golang) or external dataplane driver.
//
// Starting the background processing goroutines, which load and keep in sync with the
// state from the datastore, the "calculation graph".
//
// Starting the usage reporting and prometheus metrics endpoint threads (if configured).
//
// Then, it defers to monitorAndManageShutdown(), which blocks until one of the components
// fails, then attempts a graceful shutdown.  At that point, all the processing is in
// background goroutines.
//
// To avoid having to maintain rarely-used code paths, Felix handles updates to its
// main config parameters by exiting and allowing itself to be restarted by the init
// daemon.
func Run(configFile string, gitVersion string, buildDate string, gitRevision string) {
	// Go's RNG is not seeded by default.  Do that now.
	rand.Seed(time.Now().UTC().UnixNano())

	// Special-case handling for environment variable-configured logging:
	// Initialise early so we can trace out config parsing.
	logutils.ConfigureEarlyLogging()

	ctx := context.Background()

	if os.Getenv("GOGC") == "" {
		// Tune the GC to trade off a little extra CPU usage for significantly lower
		// occupancy at high scale.  This is worthwhile because Felix runs per-host so
		// any occupancy improvement is multiplied by the number of hosts.
		log.Debugf("No GOGC value set, defaulting to %d%%.", defaultGCPercent)
		debug.SetGCPercent(defaultGCPercent)
	}

	if len(buildinfo.GitVersion) == 0 && len(gitVersion) != 0 {
		buildinfo.GitVersion = gitVersion
		buildinfo.BuildDate = buildDate
		buildinfo.GitRevision = gitRevision
	}

	buildInfoLogCxt := log.WithFields(log.Fields{
		"version":    buildinfo.GitVersion,
		"release":    "CNX",
		"builddate":  buildinfo.BuildDate,
		"gitcommit":  buildinfo.GitRevision,
		"GOMAXPROCS": runtime.GOMAXPROCS(0),
	})
	buildInfoLogCxt.Info("Felix starting up")

	// Health monitoring, for liveness and readiness endpoints.  The following loop can take a
	// while before the datastore reports itself as ready - for example when there is data that
	// needs to be migrated from a previous version - and we still want Felix to report
	// itself as live (but not ready) while we are waiting for that.  So we create the
	// aggregator upfront and will start serving health status over HTTP as soon as we see _any_
	// config that indicates that.
	healthAggregator := health.NewHealthAggregator()

	const healthName = "felix-startup"

	// Register this function as a reporter of liveness and readiness, with no timeout.
	healthAggregator.RegisterReporter(healthName, &health.HealthReport{Live: true, Ready: true}, 0)

	// Log out the kubernetes server details that we use in BPF mode.
	log.WithFields(log.Fields{
		"KUBERNETES_SERVICE_HOST": os.Getenv("KUBERNETES_SERVICE_HOST"),
		"KUBERNETES_SERVICE_PORT": os.Getenv("KUBERNETES_SERVICE_PORT"),
	}).Info("Kubernetes server override env vars.")

	// Load the configuration from all the different sources including the
	// datastore and merge. Keep retrying on failure.  We'll sit in this
	// loop until the datastore is ready.
	log.Info("Loading configuration...")
	var backendClient bapi.Client
	var v3Client client.Interface
	var datastoreConfig apiconfig.CalicoAPIConfig
	var configParams *config.Config
	var typhaAddr string
	var numClientsCreated int
	var k8sClientSet *kubernetes.Clientset
configRetry:
	for {
		if numClientsCreated > 60 {
			// If we're in a restart loop, periodically exit (so we can be restarted) since
			// - it may solve the problem if there's something wrong with our process
			// - it prevents us from leaking connections to the datastore.
			exitWithCustomRC(configChangedRC, "Restarting to avoid leaking datastore connections")
		}

		// Make an initial report that says we're live but not yet ready.
		healthAggregator.Report(healthName, &health.HealthReport{Live: true, Ready: false})

		// Load locally-defined config, including the datastore connection
		// parameters. First the environment variables.
		configParams = config.New()
		envConfig := config.LoadConfigFromEnvironment(os.Environ())
		// Then, the config file.
		log.Infof("Loading config file: %v", configFile)
		fileConfig, err := config.LoadConfigFile(configFile)
		if err != nil {
			log.WithError(err).WithField("configFile", configFile).Error(
				"Failed to load configuration file")
			time.Sleep(1 * time.Second)
			continue configRetry
		}
		// Parse and merge the local config.
		_, err = configParams.UpdateFrom(envConfig, config.EnvironmentVariable)
		if err != nil {
			log.WithError(err).WithField("configFile", configFile).Error(
				"Failed to parse configuration environment variable")
			time.Sleep(1 * time.Second)
			continue configRetry
		}
		_, err = configParams.UpdateFrom(fileConfig, config.ConfigFile)
		if err != nil {
			log.WithError(err).WithField("configFile", configFile).Error(
				"Failed to parse configuration file")
			time.Sleep(1 * time.Second)
			continue configRetry
		}

		// Each time round this loop, check that we're serving health reports if we should
		// be, or cancel any existing server if we should not be serving any more.
		healthAggregator.ServeHTTP(configParams.HealthEnabled, configParams.HealthHost, configParams.HealthPort)

		// We should now have enough config to connect to the datastore
		// so we can load the remainder of the config.
		datastoreConfig = configParams.DatastoreConfig()
		// Can't dump the whole config because it may have sensitive information...
		log.WithField("datastore", datastoreConfig.Spec.DatastoreType).Info("Connecting to datastore")
		v3Client, err = client.New(datastoreConfig)
		if err != nil {
			log.WithError(err).Error("Failed to create datastore client")
			time.Sleep(1 * time.Second)
			continue configRetry
		}
		log.Info("Created datastore client")
		numClientsCreated++
		backendClient = v3Client.(interface{ Backend() bapi.Client }).Backend()
		for {
			globalConfig, hostConfig, err := loadConfigFromDatastore(
				ctx, backendClient, datastoreConfig, configParams.FelixHostname)
			if err == ErrNotReady {
				log.Warn("Waiting for datastore to be initialized (or migrated)")
				time.Sleep(1 * time.Second)
				healthAggregator.Report(healthName, &health.HealthReport{Live: true, Ready: true})
				continue
			} else if err != nil {
				log.WithError(err).Error("Failed to get config from datastore")
				time.Sleep(1 * time.Second)
				continue configRetry
			}
			_, err = configParams.UpdateFrom(globalConfig, config.DatastoreGlobal)
			if err != nil {
				log.WithError(err).Error("Failed update global config from datastore")
				time.Sleep(1 * time.Second)
				continue configRetry
			}
			_, err = configParams.UpdateFrom(hostConfig, config.DatastorePerHost)
			if err != nil {
				log.WithError(err).Error("Failed update host config from datastore")
				time.Sleep(1 * time.Second)
				continue configRetry
			}
			break
		}
		err = configParams.Validate()
		if err != nil {
			log.WithError(err).Error("Failed to parse/validate configuration from datastore.")
			time.Sleep(1 * time.Second)
			continue configRetry
		}

		// List all IP pools and feed them into an EncapsulationCalculator to determine if
		// IPIP and/or VXLAN encapsulations should be enabled
		ippoolKVPList, err := backendClient.List(ctx, model.ResourceListOptions{Kind: apiv3.KindIPPool}, "")
		if err != nil {
			log.WithError(err).Error("Failed to list IP Pools")
			time.Sleep(1 * time.Second)
			continue configRetry
		}
		encapCalculator := calc.NewEncapsulationCalculator(configParams, ippoolKVPList)
		configParams.Encapsulation.IPIPEnabled = encapCalculator.IPIPEnabled()
		configParams.Encapsulation.VXLANEnabled = encapCalculator.VXLANEnabled()

		// We now have some config flags that affect how we configure the syncer.
		// After loading the config from the datastore, reconnect, possibly with new
		// config.  We don't need to re-load the configuration _again_ because the
		// calculation graph will spot if the config has changed since we were initialised.
		datastoreConfig = configParams.DatastoreConfig()
		backendClient, err = backend.NewClient(datastoreConfig)
		if err != nil {
			log.WithError(err).Error("Failed to (re)connect to datastore")
			time.Sleep(1 * time.Second)
			continue configRetry
		}
		numClientsCreated++

		// Try to get a Kubernetes client.  This is needed for discovering Typha and for the BPF mode of the dataplane.
		k8sClientSet = nil
		if kc, ok := backendClient.(*k8s.KubeClient); ok {
			// Opportunistically share the k8s client with the datastore driver.  This is the best option since
			// it reduces the number of connections and it lets us piggy-back on the datastore driver's config.
			log.Info("Using Kubernetes datastore driver, sharing Kubernetes client with datastore driver.")
			k8sClientSet = kc.ClientSet
		} else {
			// Not using KDD, fall back on trying to get a Kubernetes client from the environment.
			log.Info("Not using Kubernetes datastore driver, trying to get a Kubernetes client...")
			k8sconf, err := rest.InClusterConfig()
			if err != nil {
				log.WithError(err).Info("Kubernetes in-cluster config not available. " +
					"Assuming we're not in a Kubernetes deployment.")
			} else {
				k8sClientSet, err = kubernetes.NewForConfig(k8sconf)
				if err != nil {
					log.WithError(err).Error("Got in-cluster config but failed to create Kubernetes client.")
					time.Sleep(1 * time.Second)
					continue configRetry
				}
			}
		}

		// If we're configured to discover Typha, do that now so we can retry if we fail.
		typhaAddr, err = discoverTyphaAddr(configParams, k8sClientSet)
		if err != nil {
			log.WithError(err).Error("Typha discovery enabled but discovery failed.")
			time.Sleep(1 * time.Second)
			continue configRetry
		}

		break configRetry
	}

	if numClientsCreated > 2 {
		// We don't have a way to close datastore connection so, if we reconnected after
		// a failure to load config, restart felix to avoid leaking connections.
		exitWithCustomRC(configChangedRC, "Restarting to avoid leaking datastore connections")
	}

	// Create the license monitor, which we'll use to monitor the state of the license.
	licenseMonitor := monitor.New(backendClient)
	lCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	err := licenseMonitor.RefreshLicense(lCtx)
	cancel()
	if err != nil {
		log.WithError(err).Error("Failed to get license from datastore; continuing without a license")
	}
	if configParams.DebugUseShortPollIntervals {
		log.Info("Using short license poll interval for FV")
		licenseMonitor.SetPollInterval(1 * time.Second)
	}
	// Get the license status before we adjust the config so that we can spot if the license status
	// changes below.  (If we did this read after config adjustment, we might read that the license was expired
	// after the config adjustment decided that it was valid and we'd miss the transition.)
	licenseStatus := licenseMonitor.GetLicenseStatus()

	// Correct the config based on licensed features.
	removeUnlicensedFeaturesFromConfig(configParams, licenseMonitor)

	if configParams.BPFEnabled {
		// Check for BPF dataplane support before we do anything that relies on the flag being set one way or another.
		if err := dp.SupportsBPF(); err != nil {
			log.Error("BPF dataplane mode enabled but not supported by the kernel.  Disabling BPF mode.")
			_, err := configParams.OverrideParam("BPFEnabled", "false")
			if err != nil {
				log.WithError(err).Panic("Bug: failed to override config parameter")
			}
		}
	}

	if configParams.FlowLogsCollectProcessInfo {
		if err := dp.SupportsBPFKprobe(); err != nil {
			log.Error("FlowLogsCollectProcessInfo enabled but BPF not supported by the kernel. Disabling FlowLogsCollectProcessInfo.")
			_, err := configParams.OverrideParam("FlowLogsCollectProcessInfo", "false")
			if err != nil {
				log.WithError(err).Panic("Bug: failed to override config parameter")
			}
		}
	}

	if configParams.FlowLogsCollectTcpStats {
		if err := dp.SupportsBPF(); err != nil {
			log.Error("FlowLogsCollectTcpStats enabled but BPF not supported by the kernel. Disabling FlowLogsCollectTcpStats.")
			_, err := configParams.OverrideParam("FlowLogsCollectTcpStats", "false")
			if err != nil {
				log.WithError(err).Panic("Bug: failed to override config parameter")
			}
		}
	}

	if configParams.FlowLogsCollectProcessPath {
		if !configParams.FlowLogsCollectProcessInfo {
			log.Error("FlowLogsCollectProcessPath enabled but FlowLogsCollectProcessInfo is disabled. Disabling FlowLogsCollectProcessPath.")
			_, err := configParams.OverrideParam("FlowLogsCollectProcessPath", "false")
			if err != nil {
				log.WithError(err).Panic("Bug: failed to override config parameter")
			}
		}
	}

	// We're now both live and ready.
	healthAggregator.Report(healthName, &health.HealthReport{Live: true, Ready: true})

	// Enable or disable the health HTTP server according to coalesced config.
	healthAggregator.ServeHTTP(configParams.HealthEnabled, configParams.HealthHost, configParams.HealthPort)

	// If we get here, we've loaded the configuration successfully.
	// Update log levels before we do anything else.
	logutils.ConfigureLogging(configParams)
	// Since we may have enabled more logging, log with the build context
	// again.
	buildInfoLogCxt.WithField("config", configParams).Info(
		"Successfully loaded configuration.")

	var lookupsCache *calc.LookupsCache
	var dpStatsCollector collector.Collector

	// Initialzed the lookup cache here and pass it along to both the calc_graph
	// as well as dataplane driver, which actually uses this for lookups.
	lookupsCache = calc.NewLookupsCache()

	// Start the stats collector which also depends on the lookups cache.
	dpStatsCollector = collector.New(configParams, lookupsCache, healthAggregator)

	if configParams.DebugPanicAfter > 0 {
		log.WithField("delay", configParams.DebugPanicAfter).Warn("DebugPanicAfter is set, will panic after delay!")
		go panicAfter(configParams.DebugPanicAfter)
	}

	if configParams.DebugSimulateDataRace {
		log.Warn("DebugSimulateDataRace is set, will start some racing goroutines!")
		simulateDataRace()
	}

	// We may need to temporarily disable encrypted traffic to this node in order to connect to Typha
	if configParams.WireguardEnabled {
		err := bootstrapWireguard(configParams, v3Client)
		if err != nil {
			time.Sleep(2 * time.Second) // avoid a tight restart loop
			log.WithError(err).Fatal("Couldn't bootstrap WireGuard host connectivity")
		}
	}

	// Start up the dataplane driver.  This may be the internal go-based driver or an external
	// one.
	var dpDriver dp.DataplaneDriver
	var dpDriverCmd *exec.Cmd
	var dpStopChan chan *sync.WaitGroup

	failureReportChan := make(chan string)
	configChangedRestartCallback := func() {
		failureReportChan <- reasonConfigChanged
		// It's important that we return here (rather than blocking until Felix exits).  That's because
		// some components need to shut down gracefully (e.g. the DNS cache) and we could deadlock
		// if we're being called from a place that interacts with those.
	}
	fatalErrorCallback := func(err error) {
		log.WithError(err).Error("Shutting down due to fatal error")
		failureReportChan <- reasonFatalError
		// It's important that we return here (rather than blocking until Felix exits).  That's because
		// some components need to shut down gracefully (e.g. the DNS cache) and we could deadlock
		// if we're being called from a place that interacts with those.
	}
	childExitedRestartCallback := func() { failureReportChan <- reasonChildExited }

	dpDriver, dpDriverCmd, dpStopChan = dp.StartDataplaneDriver(
		configParams.Copy(), // Copy to avoid concurrent access.
		healthAggregator,
		dpStatsCollector,
		configChangedRestartCallback,
		fatalErrorCallback,
		childExitedRestartCallback,
		v3Client.IPAM(),
		k8sClientSet,
		lookupsCache,
	)

	// Initialise the glue logic that connects the calculation graph to/from the dataplane driver.
	log.Info("Connect to the dataplane driver.")

	var connToUsageRepUpdChan chan map[string]string
	if configParams.UsageReportingEnabled {
		// Make a channel for the connector to use to send updates to the usage reporter.
		// (Otherwise, we pass in a nil channel, which disables such updates.)
		connToUsageRepUpdChan = make(chan map[string]string, 1)
	}
	dpConnector := newConnector(
		configParams.Copy(), // Copy to avoid concurrent access.
		connToUsageRepUpdChan,
		backendClient,
		v3Client,
		dpDriver,
		failureReportChan)

	// If enabled, create a server for the policy sync API.  This allows clients to connect to
	// Felix over a socket and receive policy updates.
	var policySyncServer *policysync.Server
	var policySyncProcessor *policysync.Processor
	var policySyncAPIBinder binder.Binder
	calcGraphClientChannels := []chan<- interface{}{dpConnector.ToDataplane}
	if configParams.IsLeader() && configParams.PolicySyncPathPrefix != "" {
		log.WithField("policySyncPathPrefix", configParams.PolicySyncPathPrefix).Info(
			"Policy sync API enabled.  Creating the policy sync server.")
		toPolicySync := make(chan interface{})
		policySyncUIDAllocator := policysync.NewUIDAllocator()
		policySyncProcessor = policysync.NewProcessor(configParams, toPolicySync)
		policySyncServer = policysync.NewServer(
			policySyncProcessor.JoinUpdates,
			dpStatsCollector,
			policySyncUIDAllocator.NextUID,
		)
		policySyncAPIBinder = binder.NewBinder(configParams.PolicySyncPathPrefix)
		policySyncServer.RegisterGrpc(policySyncAPIBinder.Server())
		calcGraphClientChannels = append(calcGraphClientChannels, toPolicySync)
	}

	if dpStatsCollector != nil {
		// Everybody who wanted to tweak the dpStatsCollector had a go, we can start it now!
		if err := dpStatsCollector.Start(); err != nil {
			// XXX we should panic once all dataplanes expect the collector to run.
			log.WithError(err).Panic("Stats collector did not start.")
		}
	}

	// Now create the calculation graph, which receives updates from the
	// datastore and outputs dataplane updates for the dataplane driver.
	//
	// The Syncer has its own thread and we use an extra thread for the
	// Validator, just to pipeline that part of the calculation then the
	// main calculation graph runs in a single thread for simplicity.
	// The output of the calculation graph arrives at the dataplane
	// connection via channel.
	//
	// Syncer -chan-> Validator -chan-> Calc graph -chan->   dataplane
	//        KVPair            KVPair             protobufs

	// Get a Syncer from the datastore, or a connection to our remote sync daemon, Typha,
	// which will feed the calculation graph with updates, bringing Felix into sync.
	var syncer Startable
	var typhaConnection *syncclient.SyncerClient
	syncerToValidator := calc.NewSyncerCallbacksDecoupler()
	if typhaAddr != "" {
		// Use a remote Syncer, via the Typha server.
		log.WithField("addr", typhaAddr).Info("Connecting to Typha.")
		typhaConnection = syncclient.New(
			typhaAddr,
			buildinfo.GitVersion,
			configParams.FelixHostname,
			fmt.Sprintf("Revision: %s; Build date: %s",
				buildinfo.GitRevision, buildinfo.BuildDate),
			syncerToValidator,
			&syncclient.Options{
				ReadTimeout:  configParams.TyphaReadTimeout,
				WriteTimeout: configParams.TyphaWriteTimeout,
				KeyFile:      configParams.TyphaKeyFile,
				CertFile:     configParams.TyphaCertFile,
				CAFile:       configParams.TyphaCAFile,
				ServerCN:     configParams.TyphaCN,
				ServerURISAN: configParams.TyphaURISAN,
			},
		)
	} else {
		// Use the syncer locally. If we are not including flow logs, or flow log services then there is no need
		// to watch the services and service endpoints.
		syncServices := configParams.FlowLogsFileEnabled && configParams.FlowLogsFileIncludeService
		syncServices = syncServices || configParams.TPROXYModeEnabled()
		syncer = felixsyncer.New(backendClient, datastoreConfig.Spec, syncerToValidator, syncServices, configParams.IsLeader())

		log.Info("using resource updates where applicable")
		configParams.SetUseNodeResourceUpdates(true)
	}
	log.WithField("syncer", syncer).Info("Created Syncer")

	// Start the background processing threads.
	if syncer != nil {
		log.Infof("Starting the datastore Syncer")
		syncer.Start()
	} else {
		log.Infof("Starting the Typha connection")
		err := typhaConnection.Start(context.Background())
		if err != nil {
			log.WithError(err).Error("Failed to connect to Typha. Retrying...")
			startTime := time.Now()
			for err != nil && time.Since(startTime) < 30*time.Second {
				// Set Ready to false and Live to true when unable to connect to typha
				healthAggregator.Report(healthName, &health.HealthReport{Live: true, Ready: false})
				err = typhaConnection.Start(context.Background())
				if err == nil {
					break
				}
				log.WithError(err).Debug("Retrying Typha connection")
				time.Sleep(1 * time.Second)
			}
			if err != nil {
				log.WithError(err).Fatal("Failed to connect to Typha")
			} else {
				log.Info("Connected to Typha after retries.")
				healthAggregator.Report(healthName, &health.HealthReport{Live: true, Ready: true})
			}
		}

		supportsNodeResourceUpdates, err := typhaConnection.SupportsNodeResourceUpdates(10 * time.Second)
		if err != nil {
			log.WithError(err).Error("Did not get hello message from Typha in time, assuming it does not support node resource updates")
			return
		}
		log.Debugf("Typha supports node resource updates: %v", supportsNodeResourceUpdates)
		configParams.SetUseNodeResourceUpdates(supportsNodeResourceUpdates)

		go func() {
			typhaConnection.Finished.Wait()
			failureReportChan <- "Connection to Typha failed"
		}()
	}

	// Create the ipsets/active policy calculation graph, which will
	// do the dynamic calculation of ipset memberships and active policies
	// etc.
	asyncCalcGraph := calc.NewAsyncCalcGraph(
		configParams.Copy(), // Copy to avoid concurrent access.
		licenseMonitor,
		calcGraphClientChannels,
		healthAggregator,
		lookupsCache,
	)

	// Create a stats collector to generate felix_cluster_* metrics.
	statsCollector := calc.NewStatsCollector(func(stats calc.StatsUpdate) error {
		return nil
	})
	statsCollector.RegisterWith(asyncCalcGraph.CalcGraph)

	// Create the validator, which sits between the syncer and the
	// calculation graph.
	validator := calc.NewValidationFilter(asyncCalcGraph)

	go syncerToValidator.SendTo(validator)
	asyncCalcGraph.Start()
	log.Infof("Started the processing graph")
	var stopSignalChans []chan<- *sync.WaitGroup
	if dpStopChan != nil {
		stopSignalChans = append(stopSignalChans, dpStopChan)
	}
	if configParams.EndpointReportingEnabled {
		delay := configParams.EndpointReportingDelaySecs
		log.WithField("delay", delay).Info(
			"Endpoint status reporting enabled, starting status reporter")
		dpConnector.statusReporter = statusrep.NewEndpointStatusReporter(
			configParams.FelixHostname,
			configParams.OpenstackRegion,
			dpConnector.StatusUpdatesFromDataplane,
			dpConnector.InSync,
			dpConnector.datastore,
			delay,
			delay*180,
		)
		dpConnector.statusReporter.Start()
	}

	if dpConnector.captureStatusWriter != nil {
		dpConnector.captureStatusWriter.Start()
	}

	// Start communicating with the dataplane driver.
	dpConnector.Start()

	if policySyncProcessor != nil {
		log.WithField("policySyncPathPrefix", configParams.PolicySyncPathPrefix).Info(
			"Policy sync API enabled.  Starting the policy sync server.")
		policySyncProcessor.Start()
		sc := make(chan *sync.WaitGroup)
		stopSignalChans = append(stopSignalChans, sc)
		go policySyncAPIBinder.SearchAndBind(sc)
	}

	// Send the opening message to the dataplane driver, giving it its
	// config.
	dpConnector.ToDataplane <- &proto.ConfigUpdate{
		Config: configParams.RawValues(),
	}

	if configParams.PrometheusMetricsEnabled {
		log.Info("Prometheus metrics enabled.  Starting server.")
		gaugeHost := prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "felix_host",
			Help:        "Configured Felix hostname (as a label), typically used in grouping/aggregating stats; the label defaults to the hostname of the host but can be overridden by configuration. The value of the gauge is always set to 1.",
			ConstLabels: prometheus.Labels{"host": configParams.FelixHostname},
		})
		gaugeHost.Set(1)
		prometheus.MustRegister(gaugeHost)
		go dp.ServePrometheusMetrics(configParams)
	}

	// Register signal handlers to dump memory/CPU profiles.
	logutils.RegisterProfilingSignalHandlers(configParams)

	// Monitor the license resource.  This allows us to spot
	// - license creation after we start up (triggers OnLicenseStatusChangedCallback and hence a restart)
	// - license feature updates (triggers OnFeaturesChanged and hence a restart)
	// - license expiry (triggers OnLicenseStatusChangedCallback and hence a restart)
	// License renewal with no feature changes doesn't trigger either callback.
	licenseMonitor.SetFeaturesChangedCallback(func() {
		log.Info("Active license changed, restarting...")
		failureReportChan <- reasonLicenseConfigChanged
	})

	licenseMonitor.SetStatusChangedCallback(func(newLicenseStatus lclient.LicenseStatus) {
		if licenseStatus == newLicenseStatus {
			return
		}
		// Whenever we transition between Valid/InGracePeriod/Expired/NoLicense we need to restart to
		// enable/disable features.
		log.WithFields(log.Fields{
			"oldStatus": licenseStatus,
			"newStatus": newLicenseStatus,
		}).Info("License status changed.  Restarting to enable/disable features.")
		failureReportChan <- reasonLicenseConfigChanged
	})

	// Start the license monitor, which will trigger the callback above at start of day and then whenever the license
	// status changes.
	go licenseMonitor.MonitorForever(context.Background())

	// Now monitor the worker process and our worker threads and shut
	// down the process gracefully if they fail.
	monitorAndManageShutdown(failureReportChan, dpDriverCmd, stopSignalChans)
}

func monitorAndManageShutdown(failureReportChan <-chan string, driverCmd *exec.Cmd, stopSignalChans []chan<- *sync.WaitGroup) {
	// Ask the runtime to tell us if we get a term/int signal.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGTERM)
	signal.Notify(signalChan, syscall.SIGINT)
	signal.Notify(signalChan, syscall.SIGHUP)

	// Start a background thread to tell us when the dataplane driver stops.
	// If the driver stops unexpectedly, we'll terminate this process.
	// If this process needs to stop, we'll kill the driver and then wait
	// for the message from the background thread.
	driverStoppedC := make(chan bool)
	go func() {
		if driverCmd == nil {
			log.Info("No driver process to monitor")
			return
		}
		err := driverCmd.Wait()
		log.WithError(err).Warn("Driver process stopped")
		driverStoppedC <- true
	}()

	// Wait for one of the channels to give us a reason to shut down.
	driverAlreadyStopped := driverCmd == nil
	receivedFatalSignal := false
	var reason string
	select {
	case <-driverStoppedC:
		reason = "Driver stopped"
		driverAlreadyStopped = true
	case sig := <-signalChan:
		if sig == syscall.SIGHUP {
			log.Warning("Received a SIGHUP, treating as a request to reload config")
			reason = reasonConfigChanged
		} else {
			reason = fmt.Sprintf("Received OS signal %v", sig)
			receivedFatalSignal = true
		}
	case reason = <-failureReportChan:
	}
	logCxt := log.WithField("reason", reason)
	logCxt.Warn("Felix is shutting down")

	// Keep draining the report channel so that other goroutines don't block on the channel.
	go func() {
		for msg := range failureReportChan {
			log.WithField("reason", msg).Info("Shutdown request received while already shutting down, ignoring.")
		}
	}()

	// Notify other components to stop.  Each notified component must call Done() on the wait
	// group when it has completed its shutdown.
	var stopWG sync.WaitGroup
	for _, c := range stopSignalChans {
		stopWG.Add(1)
		select {
		case c <- &stopWG:
		default:
			stopWG.Done()
		}
	}
	logCxt.Info("Told subcomponents to shut down")

	// Wait for those components to say they're done, for up to 30 seconds.
	waitC := make(chan int)
	go func() {
		stopWG.Wait()
		close(waitC)
	}()
	select {
	case <-waitC:
		logCxt.Info("Subcomponents have completed shut down")
	case <-time.After(30 * time.Second):
		logCxt.Warn("Subcomponent shut down timed out")
	}

	stopWG.Wait()

	if !driverAlreadyStopped {
		// Driver may still be running, just in case the driver is
		// unresponsive, start a thread to kill this process if we
		// don't manage to kill the driver.
		logCxt.Info("Driver still running, trying to shut it down...")
		giveUpOnSigTerm := make(chan bool)
		go func() {
			time.Sleep(4 * time.Second)
			giveUpOnSigTerm <- true
			time.Sleep(1 * time.Second)
			log.Fatal("Failed to wait for driver to exit, giving up.")
		}()
		// Signal to the driver to exit.
		err := driverCmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			logCxt.Error("failed to signal driver to exit")
		}
		select {
		case <-driverStoppedC:
			logCxt.Info("Driver shut down after SIGTERM")
		case <-giveUpOnSigTerm:
			logCxt.Error("Driver did not respond to SIGTERM, sending SIGKILL")
			_ = driverCmd.Process.Kill()
			<-driverStoppedC
			logCxt.Info("Driver shut down after SIGKILL")
		}
	}

	if !receivedFatalSignal {
		// We're exiting due to a failure or a config change, wait
		// a couple of seconds to ensure that we don't go into a tight
		// restart loop (which would make the init daemon in calico/node give
		// up trying to restart us).
		logCxt.Info("Sleeping to avoid tight restart loop.")
		go func() {
			time.Sleep(2 * time.Second)

			switch reason {
			case reasonConfigChanged, reasonLicenseConfigChanged:
				exitWithCustomRC(configChangedRC, "Exiting for config change")
				return
			case reasonEncapChanged:
				exitWithCustomRC(configChangedRC, "Exiting for encapsulation change")
				return
			case reasonChildExited:
				exitWithCustomRC(childExitedRC, "Exiting for child process exit")
				return
			}

			logCxt.Fatal("Exiting.")
		}()

		for {
			sig := <-signalChan
			if sig == syscall.SIGHUP {
				logCxt.Warning("Ignoring SIGHUP because we're already shutting down")
				continue
			}
			logCxt.WithField("signal", sig).Fatal(
				"Signal received while shutting down, exiting immediately")
		}
	}

	logCxt.Fatal("Exiting immediately")
}

func exitWithCustomRC(rc int, message string) {
	// Since log writing is done a background thread, we set the force-flush flag on this log to ensure that
	// all the in-flight logs get written before we exit.
	log.WithFields(log.Fields{
		"rc":                       rc,
		lclogutils.FieldForceFlush: true,
	}).Info(message)
	os.Exit(rc)
}

var (
	ErrNotReady = errors.New("datastore is not ready or has not been initialised")
)

func loadConfigFromDatastore(
	ctx context.Context, client bapi.Client, cfg apiconfig.CalicoAPIConfig, hostname string,
) (globalConfig, hostConfig map[string]string, err error) {

	// The configuration is split over 3 different resource types and 4 different resource
	// instances in the v3 data model:
	// -  ClusterInformation (global): name "default"
	// -  FelixConfiguration (global): name "default"
	// -  FelixConfiguration (per-host): name "node.<hostname>"
	// -  Node (per-host): name: <hostname>
	// Get the global values and host specific values separately.  We re-use the updateprocessor
	// logic to convert the single v3 resource to a set of v1 key/values.
	hostConfig = make(map[string]string)
	globalConfig = make(map[string]string)
	var ready bool
	err = getAndMergeConfig(
		ctx, client, globalConfig,
		apiv3.KindClusterInformation, "default",
		updateprocessors.NewClusterInfoUpdateProcessor(),
		&ready,
	)
	if err != nil {
		return
	}
	if !ready {
		// The ClusterInformation struct should contain the ready flag, if it is not set, abort.
		err = ErrNotReady
		return
	}
	err = getAndMergeConfig(
		ctx, client, globalConfig,
		apiv3.KindFelixConfiguration, "default",
		updateprocessors.NewFelixConfigUpdateProcessor(),
		&ready,
	)
	if err != nil {
		return
	}
	err = getAndMergeConfig(
		ctx, client, hostConfig,
		apiv3.KindFelixConfiguration, "node."+hostname,
		updateprocessors.NewFelixConfigUpdateProcessor(),
		&ready,
	)
	if err != nil {
		return
	}
	err = getAndMergeConfig(
		ctx, client, hostConfig,
		libapiv3.KindNode, hostname,
		updateprocessors.NewFelixNodeUpdateProcessor(cfg.Spec.K8sUsePodCIDR),
		&ready,
	)
	if err != nil {
		return
	}

	return
}

// getAndMergeConfig gets the v3 resource configuration extracts the separate config values
// (where each configuration value is stored in a field of the v3 resource Spec) and merges into
// the supplied map, as required by our v1-style configuration loader.
func getAndMergeConfig(
	ctx context.Context, client bapi.Client, config map[string]string,
	kind string, name string,
	configConverter watchersyncer.SyncerUpdateProcessor,
	ready *bool,
) error {
	logCxt := log.WithFields(log.Fields{"kind": kind, "name": name})

	cfg, err := client.Get(ctx, model.ResourceKey{
		Kind:      kind,
		Name:      name,
		Namespace: "",
	}, "")
	if err != nil {
		switch err.(type) {
		case cerrors.ErrorResourceDoesNotExist:
			logCxt.Info("No config of this type")
			return nil
		default:
			logCxt.WithError(err).Info("Failed to load config from datastore")
			return err
		}
	}

	// Re-use the update processor logic implemented for the Syncer.  We give it a v3 config
	// object in a KVPair and it uses the annotations defined on it to split it into v1-style
	// KV pairs.  Log any errors - but don't fail completely to avoid cyclic restarts.
	v1kvs, err := configConverter.Process(cfg)
	if err != nil {
		logCxt.WithError(err).Error("Failed to convert configuration")
	}

	// Loop through the converted values and update our config map with values from either the
	// Global or Host configs.
	for _, v1KV := range v1kvs {
		if _, ok := v1KV.Key.(model.ReadyFlagKey); ok {
			logCxt.WithField("ready", v1KV.Value).Info("Loaded ready flag")
			if v1KV.Value == true {
				*ready = true
			}
		} else if v1KV.Value != nil {
			switch k := v1KV.Key.(type) {
			case model.GlobalConfigKey:
				config[k.Name] = v1KV.Value.(string)
			case model.HostConfigKey:
				config[k.Name] = v1KV.Value.(string)
			default:
				logCxt.WithField("KV", v1KV).Debug("Skipping config - not required for initial loading")
			}
		}
	}
	return nil
}

type DataplaneConnector struct {
	config                     *config.Config
	configUpdChan              chan<- map[string]string
	ToDataplane                chan interface{}
	StatusUpdatesFromDataplane chan interface{}
	InSync                     chan bool
	failureReportChan          chan<- string
	dataplane                  dp.DataplaneDriver
	datastore                  bapi.Client
	datastorev3                client.Interface
	statusReporter             *statusrep.EndpointStatusReporter
	captureStatusWriter        *capture.StatusWriter
	captureStatusUpdates       chan *proto.PacketCaptureStatusUpdate

	datastoreInSync bool

	firstStatusReportSent bool

	wireguardStatUpdateFromDataplane chan *proto.WireguardStatusUpdate

	egressPodStatusUpdatesFromDataplane chan *proto.EgressPodStatusUpdate

	rateLimitedLogger *lclogutils.RateLimitedLogger
}

type Startable interface {
	Start()
}

func newConnector(configParams *config.Config,
	configUpdChan chan<- map[string]string,
	datastore bapi.Client,
	datastorev3 client.Interface,
	dataplane dp.DataplaneDriver,
	failureReportChan chan<- string,
) *DataplaneConnector {
	var captureStatusUpdates = make(chan *proto.PacketCaptureStatusUpdate, 100)

	felixConn := &DataplaneConnector{
		config:                              configParams,
		configUpdChan:                       configUpdChan,
		datastore:                           datastore,
		datastorev3:                         datastorev3,
		ToDataplane:                         make(chan interface{}),
		StatusUpdatesFromDataplane:          make(chan interface{}),
		InSync:                              make(chan bool, 1),
		failureReportChan:                   failureReportChan,
		dataplane:                           dataplane,
		wireguardStatUpdateFromDataplane:    make(chan *proto.WireguardStatusUpdate, 1),
		egressPodStatusUpdatesFromDataplane: make(chan *proto.EgressPodStatusUpdate, 100),
		captureStatusUpdates:                captureStatusUpdates,
		captureStatusWriter:                 capture.NewStatusWriter(configParams.FelixHostname, configParams.CaptureDir, datastorev3.PacketCaptures(), captureStatusUpdates, 2*time.Second),
		rateLimitedLogger:                   lclogutils.NewRateLimitedLogger(lclogutils.OptInterval(15 * time.Second)),
	}
	return felixConn
}

func (fc *DataplaneConnector) readMessagesFromDataplane() {
	defer func() {
		fc.shutDownProcess("Failed to read messages from dataplane")
	}()
	log.Info("Reading from dataplane driver pipe...")
	ctx := context.Background()
	for {
		payload, err := fc.dataplane.RecvMessage()
		if err != nil {
			log.WithError(err).Error("Failed to read from front-end socket")
			fc.shutDownProcess("Failed to read from front-end socket")
		}
		log.WithField("payload", payload).Debug("New message from dataplane")
		switch msg := payload.(type) {
		case *proto.ProcessStatusUpdate:
			fc.handleProcessStatusUpdate(ctx, msg)
		case *proto.WorkloadEndpointStatusUpdate:
			if fc.statusReporter != nil {
				fc.StatusUpdatesFromDataplane <- msg
			}
		case *proto.WorkloadEndpointStatusRemove:
			if fc.statusReporter != nil {
				fc.StatusUpdatesFromDataplane <- msg
			}
		case *proto.HostEndpointStatusUpdate:
			if fc.statusReporter != nil {
				fc.StatusUpdatesFromDataplane <- msg
			}
		case *proto.HostEndpointStatusRemove:
			if fc.statusReporter != nil {
				fc.StatusUpdatesFromDataplane <- msg
			}
		case *proto.WireguardStatusUpdate:
			fc.wireguardStatUpdateFromDataplane <- msg
		case *proto.PacketCaptureStatusUpdate:
			if fc.captureStatusWriter != nil {
				fc.captureStatusUpdates <- msg
			}
		case *proto.EgressPodStatusUpdate:
			fc.egressPodStatusUpdatesFromDataplane <- msg
		default:
			log.WithField("msg", msg).Warning("Unknown message from dataplane")
		}
		log.Debug("Finished handling message from front-end")
	}
}

func (fc *DataplaneConnector) handleProcessStatusUpdate(ctx context.Context, msg *proto.ProcessStatusUpdate) {
	log.Debugf("Status update from dataplane driver: %v", *msg)
	statusReport := model.StatusReport{
		Timestamp:     msg.IsoTimestamp,
		UptimeSeconds: msg.Uptime,
		FirstUpdate:   !fc.firstStatusReportSent,
	}
	kv := model.KVPair{
		Key:   model.ActiveStatusReportKey{Hostname: fc.config.FelixHostname, RegionString: model.RegionString(fc.config.OpenstackRegion)},
		Value: &statusReport,
		TTL:   fc.config.ReportingTTLSecs,
	}
	applyCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	_, err := fc.datastore.Apply(applyCtx, &kv)
	cancel()
	if err != nil {
		if _, ok := err.(cerrors.ErrorOperationNotSupported); ok {
			log.Debug("Datastore doesn't support status reports.")
			return // and it won't support the last status key either.
		} else {
			log.Warningf("Failed to write status to datastore: %v", err)
		}
	} else {
		fc.firstStatusReportSent = true
	}
	kv = model.KVPair{
		Key:   model.LastStatusReportKey{Hostname: fc.config.FelixHostname, RegionString: model.RegionString(fc.config.OpenstackRegion)},
		Value: &statusReport,
	}
	applyCtx, cancel = context.WithTimeout(ctx, 2*time.Second)
	_, err = fc.datastore.Apply(applyCtx, &kv)
	cancel()
	if err != nil {
		log.Warningf("Failed to write status to datastore: %v", err)
	}
}

func (fc *DataplaneConnector) reconcileWireguardStatUpdate(dpPubKey string) error {
	// In case of a recoverable failure (ErrorResourceUpdateConflict), retry update 3 times.
	for iter := 0; iter < 3; iter++ {
		// Read node resource from datastore and compare it with the publicKey from dataplane.
		getCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		node, err := fc.datastorev3.Nodes().Get(getCtx, fc.config.FelixHostname, options.GetOptions{})
		cancel()
		if err != nil {
			switch err.(type) {
			case cerrors.ErrorResourceDoesNotExist:
				if dpPubKey != "" {
					// If the node doesn't exist but non-empty public-key need to be set.
					log.Panic("v3 node resource must exist for Wireguard.")
				} else {
					// No node with empty dataplane update implies node resource
					// doesn't need to be processed further.
					log.Debug("v3 node resource doesn't need any update")
					return nil
				}
			}
			// return error here so we can retry in some time.
			log.WithError(err).Info("Failed to read node resource")
			return err
		}

		// Check if the public-key needs to be updated.
		storedPublicKey := node.Status.WireguardPublicKey
		if storedPublicKey != dpPubKey {
			updateCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			node.Status.WireguardPublicKey = dpPubKey
			_, err := fc.datastorev3.Nodes().Update(updateCtx, node, options.SetOptions{})
			cancel()
			if err != nil {
				// check if failure is recoverable
				switch err.(type) {
				case cerrors.ErrorResourceUpdateConflict:
					log.Debug("Update conflict, retrying update")
					continue
				}
				// retry in some time.
				log.WithError(err).Info("Failed updating node resource")
				return err
			}
			log.Debugf("Updated Wireguard public-key from %s to %s", storedPublicKey, dpPubKey)
		}
		break
	}
	return nil
}

func (fc *DataplaneConnector) handleWireguardStatUpdateFromDataplane() {
	var current *proto.WireguardStatusUpdate
	var ticker *jitter.Ticker
	var retryC <-chan time.Time

	for {
		// Block until we either get an update or it's time to retry a failed update.
		select {
		case current = <-fc.wireguardStatUpdateFromDataplane:
			log.Debugf("Wireguard status update from dataplane driver: %s", current.PublicKey)
		case <-retryC:
			log.Debug("retrying failed Wireguard status update")
		}
		if ticker != nil {
			ticker.Stop()
		}

		// Try and reconcile the current wireguard status data.
		err := fc.reconcileWireguardStatUpdate(current.PublicKey)
		if err == nil {
			current = nil
			retryC = nil
			ticker = nil
		} else {
			// retry reconciling between 2-4 seconds.
			ticker = jitter.NewTicker(2*time.Second, 2*time.Second)
			retryC = ticker.C
		}
	}
}

func (fc *DataplaneConnector) reconcileEgressPodStatusUpdate(namespace, name, gatewayIP string, started, finished time.Time) error {
	// In case of a recoverable failure (ErrorResourceUpdateConflict), retry update 3 times.
	for iter := 0; iter < 3; iter++ {
		// Read workload resource from datastore and compare it with the deletionTimestamp from dataplane.
		getCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		workload, err := fc.datastorev3.WorkloadEndpoints().Get(getCtx, namespace, name, options.GetOptions{})
		cancel()
		if err != nil {
			switch err.(type) {
			case cerrors.ErrorResourceDoesNotExist:
				// No workload endpoint found, so no need to update it.
				log.WithField("workload", namespace+"/"+name).Debug("Ignoring egress gateway status update; workload doesn't exist.")
				return nil
			}
			// return error here so we can retry in some time.
			log.WithError(err).Error("failed to read workload endpoint resource")
			return err
		}

		// Check if the maintenance details need to be updated.
		if workload.Status.EgressGateway == nil {
			startTime := metav1.NewTime(time.Time{})
			finishTime := metav1.NewTime(time.Time{})
			workload.Status.EgressGateway = &libapiv3.EgressGatewayStatus{
				MaintenanceGatewayIP: "",
				MaintenanceStarted:   &startTime,
				MaintenanceFinished:  &finishTime,
			}
		}
		currentGatewayIP := workload.Status.EgressGateway.MaintenanceGatewayIP
		currentStart := workload.Status.EgressGateway.MaintenanceStarted.Time
		currentFinish := workload.Status.EgressGateway.MaintenanceFinished.Time
		if !currentStart.Equal(started) || !currentFinish.Equal(finished) || currentGatewayIP != gatewayIP {
			workload.Status.EgressGateway.MaintenanceGatewayIP = gatewayIP
			metaMaintenanceStarted := metav1.NewTime(started)
			workload.Status.EgressGateway.MaintenanceStarted = &metaMaintenanceStarted
			metaMaintenanceFinished := metav1.NewTime(finished)
			workload.Status.EgressGateway.MaintenanceFinished = &metaMaintenanceFinished

			updateCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			// Patch only Egress Gateway annotations
			ctxPatchEgressGateway := k8sresources.ContextWithPatchMode(updateCtx, k8sresources.PatchModeEgressGateway)
			_, err = fc.datastorev3.WorkloadEndpoints().Update(ctxPatchEgressGateway, workload, options.SetOptions{})
			cancel()
			if err != nil {
				// check if failure is recoverable
				switch err.(type) {
				case cerrors.ErrorResourceUpdateConflict:
					log.Debug("update conflict, retrying update")
					continue
				}
				// retry in some time.
				log.WithError(err).Info("failed updating workload endpoint resource")
				return err
			}
			log.Debugf("updated workload endpoint maintenance timestamps from %s,%s to %s,%s for workload endpoint: %s/%s using egw pod: %s ",
				currentStart,
				currentFinish,
				started,
				finished,
				namespace,
				name,
				gatewayIP,
			)
		}
		break
	}
	return nil
}

func (fc *DataplaneConnector) handleEgressPodStatusUpdateFromDataplane() {
	var update *proto.EgressPodStatusUpdate
	updates := make(map[string]*proto.EgressPodStatusUpdate)
	var ticker *jitter.Ticker
	var retryC <-chan time.Time

	for {
		// Block until we either get an update or it's time to retry a failed update.
		select {
		case update = <-fc.egressPodStatusUpdatesFromDataplane:
			log.Debugf("egress pod status update from dataplane driver: %v", update)
			updates[update.Namespace+"/"+update.Name] = update
		case <-retryC:
			log.Debug("retrying failed Egress pod status update")
		}
		if ticker != nil {
			ticker.Stop()
		}

		// Try and reconcile the current egress pod status data.
		var err error
		for _, current := range updates {
			err = fc.reconcileEgressPodStatusUpdate(
				current.Namespace,
				current.Name,
				current.Cidr,
				proto.ConvertTimestamp(current.MaintenanceStarted),
				proto.ConvertTimestamp(current.MaintenanceFinished))
			if err == nil {
				delete(updates, current.Namespace+"/"+current.Name)
			} else {
				// break on first error, so that we can retry shortly.
				fc.rateLimitedLogger.
					WithField("workload", current.Namespace+"/"+current.Name).
					WithError(err).
					Warn("failed to apply egress gateway maintenance annotations to workload.")
				break
			}
		}
		if err == nil {
			retryC = nil
			ticker = nil
		} else {
			// retry reconciling between 2-4 seconds.
			ticker = jitter.NewTicker(2*time.Second, 2*time.Second)
			retryC = ticker.C
		}
	}
}

var handledConfigChanges = set.From("CalicoVersion", "CNXVersion", "ClusterGUID", "ClusterType", "DNSCacheEpoch", "DNSExtraTTL", "WindowsDNSExtraTTL")

func (fc *DataplaneConnector) sendMessagesToDataplaneDriver() {
	defer func() {
		fc.shutDownProcess("Failed to send messages to dataplane")
	}()

	var config map[string]string
	for {
		msg := <-fc.ToDataplane
		switch msg := msg.(type) {
		case *proto.InSync:
			log.Info("Datastore now in sync.")
			if !fc.datastoreInSync {
				log.Info("Datastore in sync for first time, sending message to status reporter.")
				fc.datastoreInSync = true
				fc.InSync <- true
			}
		case *proto.ConfigUpdate:
			if config != nil {
				log.WithFields(log.Fields{
					"old": config,
					"new": msg.Config,
				}).Info("Config updated, checking whether we need to restart")
				restartNeeded := false
				for kNew, vNew := range msg.Config {
					logCxt := log.WithFields(log.Fields{"key": kNew, "new": vNew})
					if vOld, prs := config[kNew]; !prs {
						logCxt = logCxt.WithField("updateType", "add")
					} else if vNew != vOld {
						logCxt = logCxt.WithFields(log.Fields{"old": vOld, "updateType": "update"})
					} else {
						continue
					}
					if handledConfigChanges.Contains(kNew) {
						logCxt.Info("Config change can be handled without restart")
						continue
					}
					logCxt.Warning("Config change requires restart")
					restartNeeded = true
				}
				for kOld, vOld := range config {
					logCxt := log.WithFields(log.Fields{"key": kOld, "old": vOld, "updateType": "delete"})
					if _, prs := msg.Config[kOld]; prs {
						// Key was present in the message so we've handled above.
						continue
					}
					if handledConfigChanges.Contains(kOld) {
						logCxt.Info("Config change can be handled without restart")
						continue
					}
					logCxt.Warning("Config change requires restart")
					restartNeeded = true
				}

				if restartNeeded {
					fc.shutDownProcess(reasonConfigChanged)
				}
			}

			// Take a copy of the config to compare against next time.
			config = make(map[string]string)
			for k, v := range msg.Config {
				config[k] = v
			}
		case *calc.DatastoreNotReady:
			log.Warn("Datastore became unready, need to restart.")
			fc.shutDownProcess("datastore became unready")
		case *proto.Encapsulation:
			if msg.IpipEnabled != fc.config.Encapsulation.IPIPEnabled || msg.VxlanEnabled != fc.config.Encapsulation.VXLANEnabled {
				log.Warn("IPIP and/or VXLAN encapsulation changed, need to restart.")
				fc.shutDownProcess(reasonEncapChanged)
			}
		}
		if err := fc.dataplane.SendMessage(msg); err != nil {
			fc.shutDownProcess("Failed to write to dataplane driver")
		}
	}
}

func (fc *DataplaneConnector) shutDownProcess(reason string) {
	// Send a failure report to the managed shutdown thread then give it
	// a few seconds to do the shutdown.
	fc.failureReportChan <- reason
	time.Sleep(5 * time.Second)
	// The graceful shutdown failed, terminate the process.
	log.Panic("Managed shutdown failed. Panicking.")
}

func (fc *DataplaneConnector) Start() {
	// Start a background thread to write to the dataplane driver.
	go fc.sendMessagesToDataplaneDriver()

	// Start background thread to read messages from dataplane driver.
	go fc.readMessagesFromDataplane()

	// Start a background thread to handle Wireguard update to Node.
	go fc.handleWireguardStatUpdateFromDataplane()

	// Start a background thread to handle Egress updates to pod termination status.
	go fc.handleEgressPodStatusUpdateFromDataplane()
}

func discoverTyphaAddr(configParams *config.Config, k8sClientSet kubernetes.Interface) (string, error) {
	typhaDiscoveryOpts := configParams.TyphaDiscoveryOpts()
	typhaDiscoveryOpts = append(typhaDiscoveryOpts, discovery.WithKubeClient(k8sClientSet))
	return discovery.DiscoverTyphaAddr(typhaDiscoveryOpts...)
}

type featureChecker interface {
	GetFeatureStatus(feature string) bool
	GetLicenseStatus() lclient.LicenseStatus
}

// removeUnlicensedFeaturesFromConfig modifies the requested Config depending on licensed features. Values overridden
// due to license have a higher priority than other methods of config injection and therefore cannot
// be adjusted by the user.
func removeUnlicensedFeaturesFromConfig(configParams *config.Config, licenseMonitor featureChecker) {
	licenseOverrides := make(map[string]string)

	if configParams.UseInternalDataplaneDriver {
		// Check license status of various features and disable them via config if they're not allowed.
		// Furthermore, if the features are enabled through config, raise a warning log.
		if configParams.IPSecEnabled() {
			if !licenseMonitor.GetFeatureStatus(features.IPSec) {
				log.Warn("Not licensed for IPsec feature. License either invalid or expired. " +
					"Contact Tigera support or email licensing@tigera.io")
				licenseOverrides["IPSecMode"] = "none"
			} else if licenseMonitor.GetLicenseStatus() == lclient.InGracePeriod {
				log.Warn("License for IPsec feature is in grace period, forcing IPsec into allow-unsecured " +
					"traffic mode. Contact Tigera support or email licensing@tigera.io")
				licenseOverrides["IPSecAllowUnsecuredTraffic"] = "true"
			}
		}
		if configParams.PrometheusReporterEnabled &&
			!licenseMonitor.GetFeatureStatus(features.PrometheusMetrics) {
			log.Warn("Not licensed for Prometheus Metrics feature. License either invalid or expired. " +
				"Contact Tigera support or email licensing@tigera.io")

			// Set Prometheus metrics process and reporting configs to false.
			licenseOverrides["PrometheusReporterEnabled"] = "false"
		}

		// If DropActionOverride is set to non-default "DROP" and license is not applied or valid then throw a warning message.
		if configParams.DropActionOverride != "DROP" &&
			!licenseMonitor.GetFeatureStatus(features.DropActionOverride) {
			log.Warn("Not licensed for DropActionOverride feature. License either invalid or expired. " +
				"Contact Tigera support or email licensing@tigera.io")

			// Set DropActionOverride to "DROP".
			licenseOverrides["DropActionOverride"] = "DROP"
		}

		if configParams.FlowLogsFileEnabled && !licenseMonitor.GetFeatureStatus(features.FileOutputFlowLogs) {
			log.Warn("Not licensed for Flow Logs File Output feature. License either invalid or expired. " +
				"Contact Tigera support or email licensing@tigera.io")
			licenseOverrides["FlowLogsFileEnabled"] = "false"
		}

		if configParams.L7LogsFileEnabled && !licenseMonitor.GetFeatureStatus(features.FileOutputL7Logs) {
			log.Warn("Not licensed for L7 Logs File Output feature. License either invalid or expired. " +
				"Contact Tigera support or email licensing@tigera.io")
			licenseOverrides["L7LogsFileEnabled"] = "false"
		}

		if configParams.EgressIPSupport != "Disabled" && !licenseMonitor.GetFeatureStatus(features.EgressAccessControl) {
			log.Warn("Not licensed for Egress Access Control feature. License either invalid or expired. " +
				"Contact Tigera support or email licensing@tigera.io")
			licenseOverrides["EgressIPSupport"] = "Disabled"
		}
	}

	if len(licenseOverrides) > 0 {
		log.Debug("Updating config with license check overrides")
		_, err := configParams.UpdateFrom(licenseOverrides, config.DisabledByLicenseCheck)
		if err != nil {
			log.WithError(err).Fatal("Failed to set config overrides.")
		}
	}
}
