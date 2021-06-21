// Copyright (c) 2018,2021 Tigera, Inc. All rights reserved.
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
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	confdConfig "github.com/kelseyhightower/confd/pkg/config"
	confd "github.com/kelseyhightower/confd/pkg/run"
	"github.com/projectcalico/node/pkg/nodeinit"

	"github.com/sirupsen/logrus"

	felix "github.com/projectcalico/felix/daemon"
	"github.com/projectcalico/libcalico-go/lib/logutils"

	"github.com/projectcalico/node/buildinfo"
	"github.com/projectcalico/node/cmd/calico-node/bpf"
	"github.com/projectcalico/node/pkg/allocateip"
	"github.com/projectcalico/node/pkg/cni"
	"github.com/projectcalico/node/pkg/earlynetworking"
	"github.com/projectcalico/node/pkg/health"
	"github.com/projectcalico/node/pkg/metrics"
	"github.com/projectcalico/node/pkg/lifecycle/shutdown"
	"github.com/projectcalico/node/pkg/lifecycle/startup"
)

// Create a new flag set.
var flagSet = flag.NewFlagSet("Calico", flag.ContinueOnError)

// Build the set of supported flags.
var version = flagSet.Bool("v", false, "Display version")
var runFelix = flagSet.Bool("felix", false, "Run Felix")
var runBPF = flagSet.Bool("bpf", false, "Run BPF debug tool")
var runInit = flagSet.Bool("init", false, "Do privileged initialisation of a new node (mount file systems etc).")
var runStartup = flagSet.Bool("startup", false, "Do non-privileged start-up routine.")
var runShutdown = flagSet.Bool("shutdown", false, "Do shutdown routine.")
var monitorAddrs = flagSet.Bool("monitor-addresses", false, "Monitor change in node IP addresses")
var runAllocateTunnelAddrs = flagSet.Bool("allocate-tunnel-addrs", false, "Configure tunnel addresses for this node")
var allocateTunnelAddrsRunOnce = flagSet.Bool("allocate-tunnel-addrs-run-once", false, "Run allocate-tunnel-addrs in oneshot mode")
var monitorToken = flagSet.Bool("monitor-token", false, "Watch for Kubernetes token changes, update CNI config")

// Build set of supported flags for metrics.
var runBGPMetrics = flagSet.Bool("bgp-metrics", false, "Run server for BGP Prometheus metrics endpoint")

// Options for liveness checks.
var felixLive = flagSet.Bool("felix-live", false, "Run felix liveness checks")
var birdLive = flagSet.Bool("bird-live", false, "Run bird liveness checks")
var bird6Live = flagSet.Bool("bird6-live", false, "Run bird6 liveness checks")

// Options for readiness checks.
var birdReady = flagSet.Bool("bird-ready", false, "Run BIRD readiness checks")
var bird6Ready = flagSet.Bool("bird6-ready", false, "Run BIRD6 readiness checks")
var felixReady = flagSet.Bool("felix-ready", false, "Run felix readiness checks")
var bgpMetricsReady = flagSet.Bool("bgp-metrics-ready", false, "Run BGP metrics server readiness checks")

// thresholdTime is introduced for bird readiness check. Default value is 30 sec.
var thresholdTime = flagSet.Duration("threshold-time", 30*time.Second, "Threshold time for bird readiness")

// confd flags
var runConfd = flagSet.Bool("confd", false, "Run confd")
var confdRunOnce = flagSet.Bool("confd-run-once", false, "Run confd in oneshot mode")
var confdKeep = flagSet.Bool("confd-keep-stage-file", false, "Keep stage file when running confd")
var confdConfDir = flagSet.String("confd-confdir", "/etc/calico/confd", "Confd configuration directory.")
var confdCalicoConfig = flagSet.String("confd-calicoconfig", "", "Calico configuration file.")

// Early networking flags
var runEarlyNetworking = flagSet.Bool("early", false, "Do early networking setup (e.g. for a dual-homed node)")

func main() {
	// Log to stdout.  this prevents our logs from being interpreted as errors by, for example,
	// fluentd's default configuration.
	logrus.SetOutput(os.Stdout)

	// Install a hook that adds file/line no information.
	logrus.AddHook(&logutils.ContextHook{})

	// Parse the provided flags.
	err := flagSet.Parse(os.Args[1:])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Perform some validation on the parsed flags. Only one of the following may be
	// specified at a time.
	onlyOne := []*bool{version, runFelix, runStartup, runConfd, runBGPMetrics, monitorAddrs}
	oneSelected := false
	for _, o := range onlyOne {
		if oneSelected && *o {
			fmt.Println("More than one incompatible argument provided")
			os.Exit(1)
		}

		if *o {
			oneSelected = true
		}
	}

	// Check for liveness / readiness flags. Will only run checks specified by flags.
	if *felixLive || *birdReady || *bird6Ready || *felixReady || *birdLive || *bird6Live || *bgpMetricsReady {
		health.Run(*birdReady, *bird6Ready, *felixReady, *felixLive, *birdLive, *bird6Live, *bgpMetricsReady, *thresholdTime)
		os.Exit(0)
	}

	// Decide which action to take based on the given flags.
	if *version {
		fmt.Printf("Version: %s; Release Version: %s\n", startup.CNXVERSION, startup.CNXRELEASEVERSION)
		os.Exit(0)
	} else if *runFelix {
		logrus.SetFormatter(&logutils.Formatter{Component: "felix"})
		felix.Run("/etc/calico/felix.cfg", buildinfo.GitVersion, buildinfo.BuildDate, buildinfo.GitRevision)
	} else if *runBPF {
		// Command-line tools should log to stderr to avoid confusion with the output.
		logrus.SetOutput(os.Stderr)
		bpf.RunBPFCmd()
	} else if *runInit {
		logrus.SetFormatter(&logutils.Formatter{Component: "init"})
		nodeinit.Run()
	} else if *runStartup {
		logrus.SetFormatter(&logutils.Formatter{Component: "startup"})
		startup.Run()
	} else if *runShutdown {
		logrus.SetFormatter(&logutils.Formatter{Component: "shutdown"})
		shutdown.Run()
	} else if *monitorAddrs {
		logrus.SetFormatter(&logutils.Formatter{Component: "monitor-addresses"})
		startup.ConfigureLogging()
		startup.MonitorIPAddressSubnets()
	} else if *runConfd {
		logrus.SetFormatter(&logutils.Formatter{Component: "confd"})
		cfg, err := confdConfig.InitConfig(true)
		if err != nil {
			panic(err)
		}
		cfg.ConfDir = *confdConfDir
		cfg.KeepStageFile = *confdKeep
		cfg.Onetime = *confdRunOnce
		cfg.CalicoConfig = *confdCalicoConfig
		confd.Run(cfg)
	} else if *runAllocateTunnelAddrs {
		logrus.SetFormatter(&logutils.Formatter{Component: "tunnel-ip-allocator"})
		if *allocateTunnelAddrsRunOnce {
			allocateip.Run(nil)
		} else {
			allocateip.Run(make(chan struct{}))
		}
	} else if *monitorToken {
		logrus.SetFormatter(&logutils.Formatter{Component: "cni-config-monitor"})
		cni.Run()
	} else if *runBGPMetrics {
		logrus.SetFormatter(&logutils.Formatter{Component: "bgp-metrics"})
		// To halt the metrics process, close the signal
		signal := make(chan struct{})
		metrics.Run(signal)
	} else if *runEarlyNetworking {
		logrus.SetFormatter(&logutils.Formatter{Component: "early-networking"})
		earlynetworking.Run()
	} else {
		fmt.Println("No valid options provided. Usage:")
		flagSet.PrintDefaults()
		os.Exit(1)
	}
}
