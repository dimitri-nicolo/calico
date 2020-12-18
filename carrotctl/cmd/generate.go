package cmd

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/square/go-jose.v2/jwt"

	"path/filepath"

	"github.com/tigera/licensing/client"
	"github.com/tigera/licensing/client/features"
	"github.com/tigera/licensing/datastore"
)

var (
	claims client.LicenseClaims

	customerFlag, expFlag, nodeFlag, graceFlag, debugFlag, privKeyPathFlag, certPathFlag, packageFlags *pflag.FlagSet

	// Tigera private key location.
	// Defaults to "./tigera.io_private_key.pem"
	privKeyPath string

	// Tigera license signing certificate path.
	// Defaults to "./tigera.io_certificate.pem"
	certPath string

	licensePackage              string
	absPrivKeyPath, absCertPath string

	debug = false

	exp string

	nodes int
)

func init() {
	customerFlag = GenerateLicenseCmd.PersistentFlags()
	customerFlag.StringVarP(&claims.Customer, "customer", "c", "", "Customer name")

	expFlag = GenerateLicenseCmd.PersistentFlags()
	expFlag.StringVarP(&exp, "expiry", "e", "", "License expiration date in MM/DD/YYYY format. Expires at the end of the day cluster local timezone.")

	nodeFlag = GenerateLicenseCmd.PersistentFlags()
	nodeFlag.IntVarP(&nodes, "nodes", "n", 0, "Number of nodes customer is licensed for. If not specified, it'll be an unlimited nodes license.")

	graceFlag = GenerateLicenseCmd.PersistentFlags()
	graceFlag.IntVarP(&claims.GracePeriod, "graceperiod", "g", 90, "Number of days the cluster will keep working after the license expires")

	debugFlag = GenerateLicenseCmd.PersistentFlags()
	debugFlag.BoolVar(&debug, "debug", false, "Print debug logs while generating this license")

	privKeyPathFlag = GenerateLicenseCmd.PersistentFlags()
	privKeyPathFlag.StringVar(&privKeyPath, "signing-key", "./tigera.io_private_key.pem", "Private key path to sign the license content")

	certPathFlag = GenerateLicenseCmd.PersistentFlags()
	certPathFlag.StringVar(&certPath, "certificate", "./tigera.io_certificate.pem", "Licensing intermediate certificate path")

	packageFlags = GenerateLicenseCmd.PersistentFlags()
	packageFlags.StringVarP(&licensePackage, "package", "p", features.Enterprise, "License Package and feature selection to be assigned to a license")

	GenerateLicenseCmd.MarkPersistentFlagRequired("customer")
	GenerateLicenseCmd.MarkPersistentFlagRequired("expiry")
}

var GenerateLicenseCmd = &cobra.Command{
	Use:        "generate",
	Aliases:    []string{"gen", "gen-lic", "generate-license", "make-me-a-license"},
	SuggestFor: []string{"gen", "generat", "generate-license"},
	Short:      "Generate CNX license file and store the fields in the database",
	Run: func(cmd *cobra.Command, args []string) {
		// Lower case customer name for consistency.
		claims.Customer = strings.ToLower(claims.Customer)

		// Replace spaces with '-' so the generated file name doesn't have spaces in the name.
		claims.Customer = strings.Replace(claims.Customer, " ", "-", -1)

		// Parse expiration date into time format and set it to end of the day for that date.
		claims.Expiry = parseExpiryDate(exp)

		// Generate a random UUID for the licenseID.
		claims.LicenseID = uuid.NewV4().String()

		// If the nodes flag is specified then set the value here
		// else leave it to nil (default) - which means unlimited nodes license.
		if nodeFlag.Changed("nodes") {
			claims.Nodes = &nodes
		}

		// License claims version 1.
		claims.Version = "1"

		// License all the features in accordance to a license package.
		if !features.IsValidPackageName(licensePackage) {
			log.Fatalf("[ERROR] License Package must match one of %#v", features.PackageNames)
		}
		claims.Features = strings.Split(licensePackage, "|")

		// This might be used in future. Or it could be used for debugging.
		claims.IssuedAt = jwt.NewNumericDate(time.Now().UTC())

		if len(claims.Customer) < 3 {
			log.Fatal("[ERROR] Customer name must be at least 3 characters long")
		}

		nodeCountStr := ""
		if claims.Nodes == nil {
			nodeCountStr = "Unlimited (site license)"
		} else {
			nodeCountStr = strconv.Itoa(*claims.Nodes)
		}

		// We don't set the CheckinInterval so it's an offline license since we don't have call-home server in v2.1.
		// This will be a flag when we have the licensing server, with default check-in interval set to a week,
		// the unit of this variable is hours.
		checkinIntervalStr := ""
		if claims.CheckinInterval == nil {
			checkinIntervalStr = "Offline license"
		} else {
			checkinIntervalStr = fmt.Sprintf("%d Hours", *claims.CheckinInterval)
		}

		fmt.Println("Confirm the license information:")
		fmt.Println("_________________________________________________________________________")
		fmt.Printf("Customer name:                  %s\n", claims.Customer)
		fmt.Printf("Number of nodes:                %s\n", nodeCountStr)
		fmt.Printf("License term expiration date:   %v\n", claims.Claims.Expiry.Time())
		fmt.Printf("Features (License Package):     %s\n", claims.Features)
		fmt.Printf("Checkin interval:               %s\n", checkinIntervalStr)
		fmt.Printf("Grace period (days):            %d\n", claims.GracePeriod)
		fmt.Printf("License ID (auto-generated):    %s\n", claims.LicenseID)
		fmt.Println("________________________________________________________________________")
		fmt.Println("\nIs the license information correct? [y/N]")

		var valid string
		fmt.Scanf("%s", &valid)

		if strings.ToLower(valid) != "y" {
			os.Exit(1)
		}

		absPrivKeyPath, err := filepath.Abs(privKeyPath)
		if err != nil {
			log.Fatalf("error getting the absolute path for '%s' : %s", privKeyPath, err)
		}

		absCertPath, err = filepath.Abs(certPath)
		if err != nil {
			log.Fatalf("error getting the absolute path for '%s' : %s", certPath, err)
		}

		lic, err := client.GenerateLicenseFromClaims(claims, absPrivKeyPath, absCertPath)
		if err != nil {
			log.Fatalf("error generating license from claims: %s", err)
		}

		if debug {
			fmt.Printf("Connecting to: '%s'\n", datastore.DSN)
		}

		// Store the license in the license database.
		db, err := datastore.NewDB(datastore.DSN)
		if err != nil {
			log.Fatalf("error connecting to license database: %s", err)
		}

		// Find or create the Company entry for the license.
		companyID, err := db.GetCompanyIdByName(claims.Customer)
		if err == sql.ErrNoRows {
			// Confirm creation of company with the user in case they mistyped.
			fmt.Printf("Customer '%s' not found in company database.  Create new company? [y/N]\n", claims.Customer)
			var create string
			fmt.Scanf("%s", &create)

			if strings.ToLower(create) != "y" {
				os.Exit(1)
			}

			companyID, err = db.CreateCompany(claims.Customer)
			if err != nil {
				log.Fatalf("error creating company: %s", err)
			}
		} else if err != nil {
			log.Fatalf("error looking up company: %s", err)
		}

		// Save the license in the DB.
		licenseID, err := db.CreateLicense(lic, companyID, &claims)
		if err != nil {
			log.Fatalf("error saving license to database: %s", err)
		}

		// License successfully stored in database: emit yaml file.
		err = WriteYAML(*lic, claims.Customer)
		if err != nil {
			// Remove the license from the database (leave the company around).
			cleanupErr := db.DeleteLicense(licenseID)
			if cleanupErr != nil {
				log.Fatalf("error creating the license file: %s and error cleaning license up from database: %s",
					err,
					cleanupErr,
				)
			}
			log.Fatalf("error creating the license file: %s", err)
		}

		if debug {
			spew.Dump(claims)
		}
	},
}

func parseExpiryDate(dateStr string) jwt.NumericDate {
	expSlice := strings.Split(dateStr, "/")
	if len(expSlice) != 3 {
		log.Fatal("[ERROR] expiration date must be in MM/DD/YYYY format")
	}
	yyyy, err := strconv.Atoi(expSlice[2])
	if err != nil {
		log.Fatalf("[ERROR] invalid year\n")
	}

	mm, err := strconv.Atoi(expSlice[0])
	if err != nil || mm < 1 || mm > 12 {
		log.Fatalf("[ERROR] invalid month\n")
	}

	dd, err := strconv.Atoi(expSlice[1])
	if err != nil || dd < 1 || dd > 31 {
		log.Fatalf("[ERROR] invalid date\n")
	}

	if yyyy < time.Now().Year() {
		log.Fatalf("[ERROR] Year cannot be in the past! Unless you're a time traveller, in which case go back in time and stop me from writing this validation :P")
	}

	return jwt.NewNumericDate(time.Date(yyyy, time.Month(mm), dd, 23, 59, 59, 999999999, time.Local))
}
