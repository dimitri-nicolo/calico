// Copyright (c) 2024 Tigera, Inc. All rights reserved.

package config

import "os"

var (
	TLSCert, TLSKey       string
	EnvoyImg, DikastesImg string
)

func init() {
	TLSCert = os.Getenv("L7ADMCTRL_TLSCERTPATH")
	TLSKey = os.Getenv("L7ADMCTRL_TLSKEYPATH")
	EnvoyImg = os.Getenv("L7ADMCTRL_ENVOYIMAGE")
	DikastesImg = os.Getenv("L7ADMCTRL_DIKASTESIMAGE")

	missingRequired := TLSCert == "" ||
		TLSKey == "" ||
		EnvoyImg == "" ||
		DikastesImg == ""
	if missingRequired {
		panic("Required env vars not declared.")
	}
}
