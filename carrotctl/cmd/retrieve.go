package cmd

import (
	"log"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/tigera/licensing/client"
	cryptolicensing "github.com/tigera/licensing/crypto"
	"github.com/tigera/licensing/datastore"

	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

var (
	retrieveUUIDFlag *pflag.FlagSet
	retrieveUUID     string

	retrieveCertPathFlag *pflag.FlagSet
	retrieveCertPath     string
)

func init() {
	retrieveUUIDFlag = RetrieveLicenseCmd.PersistentFlags()
	retrieveUUIDFlag.StringVarP(&retrieveUUID, "license-uuid", "u", "", "License UUID")
	RetrieveLicenseCmd.MarkPersistentFlagRequired("license-uuid")

	retrieveCertPathFlag = RetrieveLicenseCmd.PersistentFlags()
	retrieveCertPathFlag.StringVar(&retrieveCertPath, "certificate", "./tigera.io_certificate.pem", "Licensing intermediate certificate path")
}

var RetrieveLicenseCmd = &cobra.Command{
	Use:        "retrieve a previously generated license from the database",
	Aliases:    []string{"retrieve", "retrieve-license"},
	SuggestFor: []string{"ret", "regenerate"},
	Short:      "Retrieve a license",
	Run: func(cmd *cobra.Command, args []string) {

		if len(retrieveUUID) != 36 {
			log.Fatal("[ERROR] License UUID must be 36 characters long")
		}

		// Connect to the license database.
		db, err := datastore.NewDB(datastore.DSN)
		if err != nil {
			log.Fatalf("error connecting to license database: %s", err)
		}

		// Get the license.
		licenseInfo, err := db.GetLicenseByUUID(retrieveUUID)
		if err != nil {
			log.Fatalf("error getting license: %s", err)
		}

		absCertPath, err := filepath.Abs(retrieveCertPath)
		if err != nil {
			log.Fatalf("error getting the absolute path for '%s' : %s", retrieveCertPath, err)
		}

		// Regenerate the license.
		lic := api.NewLicenseKey()
		lic.Name = client.ResourceName
		lic.Spec.Token = licenseInfo.JWT
		lic.Spec.Certificate = cryptolicensing.ReadCertPemFromFile(absCertPath)

		// License successfully stored in database: emit yaml file.
		err = WriteYAML(*lic, retrieveUUID)
		if err != nil {
			log.Fatalf("error creating the license file: %s", err)
		}
	},
}
