package cmd

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/satori/go.uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/tigera/licensing/client"
	"path/filepath"
)

var (
	claims client.LicenseClaims

	customerFlag, expFlag, nodeFlag, graceFlag, debugFlag, privKeyPathFlag, certPathFlag *pflag.FlagSet

	// Tigera private key location.
	// Defaults to "./tigera.io_private_key.pem"
	privKeyPath string

	// Tigera license signing certificate path.
	// Defaults to "./tigera.io_certificate.pem"
	certPath string

	// allFeaturesV21 is to indicate all the features available in CNX v2.1.
	// We don't license individual features in v2.1 so this is not checked on the client side.
	allFeaturesV21 = []string{"cnx", "all"}

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

	GenerateLicenseCmd.MarkPersistentFlagRequired("customer")
	GenerateLicenseCmd.MarkPersistentFlagRequired("expiry")
}

var GenerateLicenseCmd = &cobra.Command{
	Use: "generate",
	Aliases: []string{"gen", "gen-lic", "generate-license", "make-me-a-license"},
	SuggestFor: []string{"gen", "generat", "generate-license"},
	Short: "Generate tigera CNX license file",
	Run: func(cmd *cobra.Command, args []string) {

		// Parse expiration date into time format and set it to end of the day for that date.
		claims.Expiry = parseExpiryDate(exp)

		// Generate a random UUID for the licenseID.
		claims.LicenseID = uuid.NewV4().String()

		// If the nodes flag is specified then set the value here
		// else leave it to nil (default) - which means unlimited nodes license.
		if nodeFlag.Changed("nodes"){
			claims.Nodes = &nodes
		}

		// This might be configurable in future. Right now we just license all the features.
		claims.Features = allFeaturesV21

		// This might be used in future. Or it could be used for debugging.
		claims.IssuedAt = jwt.NewNumericDate(time.Now().UTC())

		if len(claims.Customer) < 3 {
			log.Fatal("[ERROR] Customer name must be at least 3 charecters long")
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
		fmt.Printf("Features:                       %s\n", claims.Features)
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

		absCertPath, _ = filepath.Abs(certPath)
		if err != nil {
			log.Fatalf("error getting the absolute path for '%s' : %s", certPath, err)
		}

		lic, err := client.GenerateLicenseFromClaims(claims, absPrivKeyPath, absCertPath)
		if err != nil {
			log.Fatalf("error generating license from claims: %s", err)
		}

		err = WriteYAML(*lic, claims.Customer)
		if err != nil {
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
