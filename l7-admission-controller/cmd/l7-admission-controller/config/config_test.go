// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package config

import (
	"os"
	"testing"
)

func TestFromEnv(t *testing.T) {
	// Test case 1: Environment variables are set
	os.Setenv("L7ADMCTRL_TLSCERTPATH", "/path/to/tls/cert.pem")
	os.Setenv("L7ADMCTRL_TLSKEYPATH", "/path/to/tls/key.pem")
	os.Setenv("L7ADMCTRL_ENVOYIMAGE", "envoy:v1.2.3")
	os.Setenv("L7ADMCTRL_DIKASTESIMAGE", "dikastes:v4.5.6")

	config, err := FromEnv()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedConfig := &Config{
		TLSCert:     "/path/to/tls/cert.pem",
		TLSKey:      "/path/to/tls/key.pem",
		EnvoyImg:    "envoy:v1.2.3",
		DikastesImg: "dikastes:v4.5.6",
	}

	if config.TLSCert != expectedConfig.TLSCert ||
		config.TLSKey != expectedConfig.TLSKey ||
		config.EnvoyImg != expectedConfig.EnvoyImg ||
		config.DikastesImg != expectedConfig.DikastesImg {
		t.Fatalf("Unexpected config values. Got: %+v, Expected: %+v", config, expectedConfig)
	}

	// Test case 2: Environment variables are not set
	os.Unsetenv("L7ADMCTRL_TLSCERTPATH")
	os.Unsetenv("L7ADMCTRL_TLSKEYPATH")
	os.Unsetenv("L7ADMCTRL_ENVOYIMAGE")
	os.Unsetenv("L7ADMCTRL_DIKASTESIMAGE")

	_, err = FromEnv()
	if err == nil {
		t.Error("Expected error, but got nil")
	}
}
