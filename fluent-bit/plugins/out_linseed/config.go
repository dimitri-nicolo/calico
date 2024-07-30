// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

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

	skipVerify := false
	if b, err := strconv.ParseBool(os.Getenv("TLS_VERIFY")); err == nil {
		skipVerify = !b
	}
	logrus.Debugf("skip_verify=%v", skipVerify)

	serviceAccountName, err := getServiceAccountName(kubeconfig)
	if err != nil {
		return nil, err
	}

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

		endpoint:           output.FLBPluginConfigKey(plugin, "Endpoint"),
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
