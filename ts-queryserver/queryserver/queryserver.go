// Copyright (c) 2018-2023 Tigera, Inc. All rights reserved.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/kelseyhightower/envconfig"

	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/lma/pkg/k8s"
	"github.com/projectcalico/calico/ts-queryserver/pkg/clientmgr"
	authjwt "github.com/projectcalico/calico/ts-queryserver/queryserver/auth"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/config"
	handler "github.com/projectcalico/calico/ts-queryserver/queryserver/handlers/auth"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/server"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Client Config from environment option.
const cfFromEnv = ""

// These are filled out during the build process (using git describe output)
var VERSION, BUILD_DATE, GIT_DESCRIPTION, GIT_REVISION string
var version bool

func PrintVersion() error {
	fmt.Println("Version:     ", VERSION)
	fmt.Println("Build date:  ", BUILD_DATE)
	fmt.Println("Git tag ref: ", GIT_DESCRIPTION)
	fmt.Println("Git commit:  ", GIT_REVISION)
	return nil
}

func init() {
	// Add a flag to check the version.
	flag.BoolVar(&version, "version", false, "Display version")
}

func main() {
	flag.Parse()
	if version {
		_ = PrintVersion()
		os.Exit(0)
	}

	// Set the logging level (default to warning).
	logLevel := log.WarnLevel
	logLevelStr := os.Getenv("LOGLEVEL")
	if logLevelStr != "" {
		logLevel = logutils.SafeParseLogLevel(logLevelStr)
	}
	log.SetLevel(logLevel)

	// Load the client config. Currently, we only support loading from environment.
	cfg, err := clientmgr.LoadClientConfig(cfFromEnv)
	if err != nil {
		log.Error("Error loading config")
	}
	log.Infof("Loaded client config: %#v", cfg.Spec)

	// Get server config from environments.
	serverCfg := &config.Config{}
	err = envconfig.Process("", serverCfg)
	if err != nil {
		log.WithError(err).Fatal("Error getting server config")
	}

	// Create a rest config from supplied kubeconfig.
	kubeconfig := os.Getenv("KUBECONFIG")
	restCfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.WithError(err).Fatal("Error processing kubeconfig file in environment variable KUBECONFIG")
	}
	restCfg.Timeout = 15 * time.Second

	// Create a k8s and calico v3 clientset, and associated informer factories.
	k8sClient, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		log.WithError(err).Fatal("Failed to load k8s client")
	}

	// Define a new authentication handler.
	authJWT, err := authjwt.GetJWTAuth(serverCfg, restCfg, k8sClient)
	if err != nil {
		log.WithError(err).Fatal("Failed to create authenticator")
	}
	authnHandler := handler.NewAuthHandler(authJWT)

	// Define a new authorization handler
	k8sClientSet := k8s.NewClientSetFactory("", "")
	authzHandler := authjwt.NewAuthorizer(k8sClientSet)

	// Start the server.
	srv := server.NewServer(k8sClient, cfg, serverCfg, authnHandler, authzHandler)
	if err := srv.Start(); err != nil {
		log.WithError(err).Fatal("Error starting queryserver")
	}

	// Wait while the server is running.
	srv.Wait()
}
