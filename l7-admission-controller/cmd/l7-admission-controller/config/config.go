// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package config

import (
	"fmt"
	"os"
)

type Config struct {
	TLSCert, TLSKey       string
	EnvoyImg, DikastesImg string
}

func FromEnv() (*Config, error) {
	tlsCert := os.Getenv("L7ADMCTRL_TLSCERTPATH")
	tlsKey := os.Getenv("L7ADMCTRL_TLSKEYPATH")
	envoyImg := os.Getenv("L7ADMCTRL_ENVOYIMAGE")
	dikastesImg := os.Getenv("L7ADMCTRL_DIKASTESIMAGE")

	for _, v := range []string{tlsCert, tlsKey, envoyImg, dikastesImg} {
		if v == "" {
			return nil, fmt.Errorf("one of required env vars not declared")
		}
	}

	return &Config{
		TLSCert:     tlsCert,
		TLSKey:      tlsKey,
		EnvoyImg:    envoyImg,
		DikastesImg: dikastesImg,
	}, nil
}
