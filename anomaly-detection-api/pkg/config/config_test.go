// Copyright (c) 2022 Tigera. All rights reserved.
package config_test

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/anomaly-detection-api/pkg/config"
)

var _ = Describe("Config test", func() {

	It("should set default values on all Config fields if no env config is provided", func() {
		config, err := config.NewConfigFromEnv()

		Expect(err).To(BeNil())
		Expect(config.ListenAddr).To(Equal(":8080"))
		Expect(config.ServiceEndpoint).To(Equal("http://localhost:8080"))
		Expect(config.HostedNamespace).To(Equal("tigera-intrusion-detection"))
		Expect(config.StoragePath).To(Equal("/store"))

		Expect(config.TLSCert).To(Equal("/tls/tls.crt"))
		Expect(config.TLSKey).To(Equal("/tls/tls.key"))

		Expect(config.DebugRunWithRBACDisabled).To(Equal(false))
		Expect(config.LogLevel).To(Equal("info"))
	})

	It("should get env vars set for the corresponding Config fields", func() {
		os.Setenv("LISTEN_ADDR", ":8081")
		os.Setenv("ENDPOINT_URL", "http://anomaly-detection-api.tigera-intrusion-detection.svc.cluster.local:8080")
		os.Setenv("STORAGE_PATH", "/storage")
		os.Setenv("NAMESPACE", "another-hosted-namespace")
		os.Setenv("TLS_CERT", "/other-tls-folder/tls.crt")
		os.Setenv("TLS_KEY", "/other-tls-folder/tls.key")
		os.Setenv("DEBUG_RBAC_DISABLED", "true")
		os.Setenv("LOG_LEVEL", "debug")

		defer os.Unsetenv("LISTEN_ADDR")
		defer os.Unsetenv("ENDPOINT_URL")
		defer os.Unsetenv("STORAGE_PATH")
		defer os.Unsetenv("NAMESPACE")
		defer os.Unsetenv("TLS_CERT")
		defer os.Unsetenv("TLS_KEY")
		defer os.Unsetenv("DEBUG_RBAC_DISABLED")
		defer os.Unsetenv("LOG_LEVEL")

		config, err := config.NewConfigFromEnv()

		Expect(err).To(BeNil())
		Expect(config.ListenAddr).To(Equal(":8081"))
		Expect(config.ServiceEndpoint).To(Equal("http://anomaly-detection-api.tigera-intrusion-detection.svc.cluster.local:8080"))
		Expect(config.StoragePath).To(Equal("/storage"))
		Expect(config.HostedNamespace).To(Equal("another-hosted-namespace"))
		Expect(config.TLSCert).To(Equal("/other-tls-folder/tls.crt"))
		Expect(config.TLSKey).To(Equal("/other-tls-folder/tls.key"))
		Expect(config.DebugRunWithRBACDisabled).To(Equal(true))
		Expect(config.LogLevel).To(Equal("debug"))
	})
})
