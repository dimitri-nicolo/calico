// Copyright (c) 2023 Tigera, Inc. All rights reserved.

//go:build fvtests

package fv_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/felix/fv/containers"
	"github.com/projectcalico/calico/libcalico-go/lib/logutils"
	"github.com/projectcalico/calico/linseed/pkg/client"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
	"github.com/projectcalico/calico/linseed/pkg/config"
)

// Use a relative path to the client token in the calico-private/linseed/fv directory.
const TokenPath = "./client-token"

func NewLinseedClient(args *RunLinseedArgs) (client.Client, error) {
	cfg := rest.Config{
		CACertPath:     "cert/RootCA.crt",
		URL:            fmt.Sprintf("https://localhost:%d/", args.Port),
		ClientCertPath: "cert/localhost.crt",
		ClientKeyPath:  "cert/localhost.key",
	}

	// The token is created as part of FV setup in the Makefile, and mounted into the container that
	// runs the FV binaries.
	return client.NewClient(args.TenantID, cfg, rest.WithTokenPath(TokenPath))
}

func DefaultLinseedArgs() *RunLinseedArgs {
	return &RunLinseedArgs{
		Backend:     config.BackendTypeMultiIndex,
		TenantID:    "tenant-a",
		Port:        8443,
		MetricsPort: 9095,
		HealthPort:  8080,
	}
}

type RunLinseedArgs struct {
	Backend     config.BackendType
	TenantID    string
	Port        int
	MetricsPort int
	HealthPort  int
}

func RunLinseed(t *testing.T, args *RunLinseedArgs) *containers.Container {
	// The container library uses gomega, so we need to connect our testing.T to it.
	gomega.RegisterTestingT(t)

	// Get the current working directory, which we expect to by the fv dir.
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Turn it to an absolute path.
	cwd, err = filepath.Abs(cwd)
	require.NoError(t, err)

	// The certs path is relative to the fv dir.
	certsPath := filepath.Join(cwd, "../../hack/test/certs/")

	dockerArgs := []string{
		"--net=host",
		"-v", fmt.Sprintf("%s/cert/localhost.crt:/certs/https/tls.crt", cwd),
		"-v", fmt.Sprintf("%s/cert/localhost.key:/certs/https/tls.key", cwd),
		"-v", fmt.Sprintf("%s/cert/RootCA.crt:/certs/https/client.crt", cwd),
		"-v", fmt.Sprintf("%s/linseed-token:/var/run/secrets/kubernetes.io/serviceaccount/token", cwd),
		"-v", fmt.Sprintf("%s/ca.pem:/var/run/secrets/kubernetes.io/serviceaccount/ca.crt", certsPath),
		"-e", "KUBERNETES_SERVICE_HOST=127.0.0.1",
		"-e", "KUBERNETES_SERVICE_PORT=6443",
		"-e", "ELASTIC_HOST=localhost",
		"-e", "ELASTIC_SCHEME=http",
		"-e", "LINSEED_LOG_LEVEL=debug",
		"-e", fmt.Sprintf("LINSEED_HEALTH_PORT=%d", args.HealthPort),
		"-e", fmt.Sprintf("LINSEED_ENABLE_METRICS=%t", args.MetricsPort != 0),
		"-e", fmt.Sprintf("LINSEED_METRICS_PORT=%d", args.MetricsPort),
		"-e", fmt.Sprintf("LINSEED_PORT=%d", args.Port),
		"-e", fmt.Sprintf("LINSEED_BACKEND=%s", args.Backend),
		"-e", fmt.Sprintf("LINSEED_EXPECTED_TENANT_ID=%s", args.TenantID),
		"tigera/linseed:latest",
	}

	name := "tigera-linseed-fv"
	if args.TenantID != "" {
		name += "-" + args.TenantID
	}

	c := containers.Run(name, containers.RunOpts{AutoRemove: true, OutputWriter: logutils.TestingTWriter{t}}, dockerArgs...)
	c.StopLogs()
	return c
}
