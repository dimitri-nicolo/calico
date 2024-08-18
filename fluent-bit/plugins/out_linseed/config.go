// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

import "C"

type Config struct {
	clientset *kubernetes.Clientset

	endpoint           string
	insecureSkipVerify bool

	serviceAccountName string
	expiration         time.Time
	token              string
}

func NewConfig(plugin unsafe.Pointer) (*Config, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = output.FLBPluginConfigKey(plugin, "Kubeconfig")
	}
	logrus.Debugf("read kubeconfig from %q", kubeconfig)

	endpoint := os.Getenv("ENDPOINT")
	if endpoint == "" {
		endpoint = output.FLBPluginConfigKey(plugin, "Endpoint")
	}
	if _, err := url.Parse(endpoint); err != nil {
		return nil, err
	}
	logrus.Debugf("log ingestion endpoint %q", endpoint)

	skipVerify := false
	tlsVerify := output.FLBPluginConfigKey(plugin, "tls.verify")
	if b, err := strconv.ParseBool(tlsVerify); err == nil {
		skipVerify = !b
	}
	logrus.Debugf("skip_verify=%v", skipVerify)

	serviceAccountName, err := getServiceAccountName(kubeconfig)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("service_account=%v", serviceAccountName)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Config{
		clientset: clientset,

		endpoint:           endpoint,
		insecureSkipVerify: skipVerify,

		serviceAccountName: serviceAccountName,
	}, nil
}

func getServiceAccountName(kubeconfig string) (string, error) {
	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig: %q", kubeconfig)
	}

	currentContext := config.CurrentContext
	if currentContext == "" {
		return "", fmt.Errorf("no current-context set in kubeconfig")
	}

	ctx, exists := config.Contexts[currentContext]
	if !exists {
		return "", fmt.Errorf("context %q not found in kubeconfig", currentContext)
	}
	return ctx.AuthInfo, nil
}
