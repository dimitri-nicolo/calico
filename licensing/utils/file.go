package utils

import (
	"log"
	"os"

	yaml "github.com/projectcalico/go-yaml-wrapper"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

// ReadFile reads license from file and returns the LicenseKey resource.
func ReadFile(path string) api.LicenseKey {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	lic := api.NewLicenseKey()
	err = yaml.Unmarshal(data, &lic)
	if err != nil {
		log.Fatalf("error unmarshaling the license data")
	}

	return *lic
}
