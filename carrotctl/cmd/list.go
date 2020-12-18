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
	customerListFlag, allFlag *pflag.FlagSet
	customerListName string
	all = false
)

func init() {
	customerListFlag = ListLicensesCmd.PersistentFlags()
	customerListFlag.StringVarP(&customerListName, "customer", "c", "", "Customer name")

	allFlag = ListLicensesCmd.PersistentFlags()
	allFlag.BoolVar(&all, "all", false, "List all companies and their licenses")
}

var ListLicensesCmd = &cobra.Command{
	Use:        "list licenses for a specific or all customers",
	Aliases:    []string{"list", "list-licenses"},
	SuggestFor: []string{"ls", "get"},
	Short:      "List licenses",
	Run: func(cmd *cobra.Command, args []string) {

		if customerListFlag.Changed("customer") && allFlag.Changed("all") {
			log.Fatalf("[ERROR] Cannot specify '--all' and '--customer' flags together")
		}

		if customerListFlag.Changed("customer") && len(customerListName) < 3 {
			log.Fatal("[ERROR] Customer name must be at least 3 characters long")
		}

		// Connect to the license database.
		db, err := datastore.NewDB(datastore.DSN)
		if err != nil {
			log.Fatalf("error connecting to license database: %s", err)
		}

		companyList := []*datastore.Company{}

		if customerListFlag.Changed("customer") {
			// Find the Company entry for the license.
			companyID, err := db.GetCompanyIdByName(customerListName)
			if err == sql.ErrNoRows {
				// Confirm creation of company with the user in case they mistyped.
				log.Fatalf("company %s not found in license database", customerListName)
			} else if err != nil {
				log.Fatalf("error looking up company: %s", err)
			}

			company, err := db.GetCompanyById(companyID)
			if err != nil {
				log.Fatalf("[ERROR] error looking up company: %s", err)
			}
			companyList = append(companyList, company)
		} else {
			// If '--all' flag is specified then get all companies and append them to the companyList.
			allComp, err := db.AllCompanies()
			if err != nil {
				log.Fatalf("[ERROR] error looking up all companies: %s", err)
			}
			companyList = append(companyList, allComp...)
		}

		fmt.Println("COMPANY                    LICENSE_ID                                 NODES          EXPIRY                          FEATURES")

		// Go through all or a specific company and list their license info.
		for _, comp := range companyList {
			// Get the licenses for that company.
			licenses, err := db.GetLicensesByCompany(comp.ID)
			if err != nil {
				log.Fatalf("error getting licenses: %s", err)
			}

			for _, lic := range licenses {
				nodes := "Unlimited"
				if lic.Nodes != nil {
					nodes = fmt.Sprintf("%9d", *lic.Nodes)
				}
				fmt.Printf("%-23s    %-40s   %-12s   %s   %s\n", comp.Name, lic.UUID, nodes, lic.Expiry, lic.Features)
			}
		}
	},
}
