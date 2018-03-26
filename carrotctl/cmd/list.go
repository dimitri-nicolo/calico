package cmd

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/tigera/licensing/datastore"
)

var (
	customerListFlag *pflag.FlagSet
	customerListName string
)

func init() {
	customerListFlag = ListLicensesCmd.PersistentFlags()
	customerListFlag.StringVarP(&customerListName, "customer", "c", "", "Customer name")
	ListLicensesCmd.MarkPersistentFlagRequired("customer")
}

var ListLicensesCmd = &cobra.Command{
	Use:        "list licenses for a customer",
	Aliases:    []string{"list", "list-licenses"},
	SuggestFor: []string{"ls", "get"},
	Short:      "List licenses for a customer",
	Run: func(cmd *cobra.Command, args []string) {

		if len(customerListName) < 3 {
			log.Fatal("[ERROR] Customer name must be at least 3 charecters long")
		}

		// Connect to the license database.
		db, err := datastore.NewDB(datastore.DSN)
		if err != nil {
			log.Fatalf("error connecting to license database: %s", err)
		}

		// Find the Company entry for the license.
		companyID, err := db.GetCompanyByName(customerListName)
		if err == sql.ErrNoRows {
			// Confirm creation of company with the user in case they mistyped.
			log.Fatalf("company %s not found in license database", customerListName)
		} else if err != nil {
			log.Fatalf("error looking up company: %s", err)
		}

		// Get the licenses for that company.
		licenses, err := db.GetLicensesByCompany(companyID)
		if err != nil {
			log.Fatalf("error getting licenses: %s", err)
		}

		fmt.Println("LICENSE UUID                           MAX NODES   EXPIRY                          FEATURES")
		for _, lic := range licenses {
			fmt.Printf("%s   %9d   %s   %s\n", lic.UUID, lic.Nodes, lic.Expiry, lic.Features)
		}
	},
}
