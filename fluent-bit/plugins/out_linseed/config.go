// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/sirupsen/logrus"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type PluginConfigKeyFunc func(pointer unsafe.Pointer, key string) string

type Config struct {
	clientset kubernetes.Interface

	endpoint           string
	insecureSkipVerify bool

	serviceAccountName string
	expiration         time.Time
	token              string

	pluginConfigKeyFn PluginConfigKeyFunc
}

func NewConfig(plugin unsafe.Pointer, fn PluginConfigKeyFunc) (*Config, error) {
	cfg := &Config{
		insecureSkipVerify: false,
		pluginConfigKeyFn:  fn,
	}

	kubeconfig := cfg.getEnvOrPluginConfig(plugin, "Kubeconfig")
	logrus.Debugf("read kubeconfig from %q", kubeconfig)

	endpoint := cfg.getEnvOrPluginConfig(plugin, "Endpoint")
	if _, err := url.Parse(endpoint); err != nil {
		return nil, err
	}
	cfg.endpoint = endpoint
	logrus.Debugf("log ingestion endpoint %q", cfg.endpoint)

	skipVerify := false
	tlsVerify := cfg.pluginConfigKeyFn(plugin, "tls.verify")
	if b, err := strconv.ParseBool(tlsVerify); err == nil {
		skipVerify = !b
	}
	cfg.insecureSkipVerify = skipVerify
	logrus.Debugf("skip_verify=%v", cfg.insecureSkipVerify)

	serviceAccountName, err := cfg.extractServiceAccountName(kubeconfig)
	if err != nil {
		return nil, err
	}
	cfg.serviceAccountName = serviceAccountName
	logrus.Debugf("service_account=%v", cfg.serviceAccountName)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	cfg.clientset = clientset

	return cfg, nil
}

func (c *Config) getEnvOrPluginConfig(plugin unsafe.Pointer, key string) string {
	val := os.Getenv(strings.ToUpper(key))
	if val == "" {
		val = cfg.pluginConfigKeyFn(plugin, key)
	}
	return val
}

func (c *Config) extractServiceAccountName(kubeconfig string) (string, error) {
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
