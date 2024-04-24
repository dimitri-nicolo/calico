// Copyright (c) 2024 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"fmt"
	"testing"

	"github.com/onsi/gomega"

	"github.com/projectcalico/calico/felix/fv/containers"
	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
)

type RunKibanaProxyArgs struct {
	Port                  int
	HealthPort            int
	ElasticEndpoint       string
	ElasticClientCertPath string
	ElasticCAPath         string
	TenantID              string
}

func DefaultKibanaProxyArgs() *RunKibanaProxyArgs {
	return &RunKibanaProxyArgs{
		Port:            5555,
		ElasticEndpoint: "http://localhost:9200",
		TenantID:        "A",
	}
}

func RunKibanaProxy(t *testing.T, args *RunKibanaProxyArgs) *containers.Container {
	// The container library uses gomega, so we need to connect our testing.T to it.
	gomega.RegisterTestingT(t)

	dockerArgs := []string{
		"--net=host",
		"-e", "ES_GATEWAY_LOG_LEVEL=TRACE",
		"-e", "ES_GATEWAY_KIBANA_CATCH_ALL_ROUTE=/",
		"-e", fmt.Sprintf("ES_GATEWAY_KIBANA_PROXY_PORT=%d", args.Port),
		"-e", fmt.Sprintf("ES_GATEWAY_ELASTIC_ENDPOINT=%s", args.ElasticEndpoint),
		"-e", fmt.Sprintf("TENANT_ID=%s", args.TenantID),
		"tigera/es-gateway:latest",
		"-run-as-kibana-proxy",
	}

	name := "tigera-kibana-proxy-fv"

	c := containers.Run(name, containers.RunOpts{AutoRemove: true, OutputWriter: logutils.TestingTWriter{t}}, dockerArgs...)
	c.StopLogs()
	return c
}

type RunKibanaArgs struct {
	Image        string
	ElasticHosts string
}

func RunKibana(t *testing.T, args *RunKibanaArgs) *containers.Container {
	// The container library uses gomega, so we need to connect our testing.T to it.
	gomega.RegisterTestingT(t)

	dockerArgs := []string{
		"--net=host",
		"-e", fmt.Sprintf("ELASTICSEARCH_HOSTS=%s", args.ElasticHosts),
		args.Image,
	}

	name := "tigera-kibana"

	c := containers.Run(name, containers.RunOpts{AutoRemove: true, OutputWriter: logutils.TestingTWriter{t}}, dockerArgs...)
	c.StopLogs()
	return c
}
