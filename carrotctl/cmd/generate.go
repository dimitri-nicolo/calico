package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/satori/go.uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/tigera/licensing/client"
)

var (
	claims client.LicenseClaims

	customerFlag, debugFlag, expFlag, nodeFlag, graceFlag *pflag.FlagSet

	// Tigera private key location.
	privKeyPath = "./tigera.io_private_key.pem"

	// Tigera license signing certificate path.
	certPath = "./tigera.io_certificate.pem"

	absPrivKeyPath, absCertPath string

	debug = false

	exp string
)

func init() {
	customerFlag = GenerateLicenseCmd.PersistentFlags()
	expFlag = GenerateLicenseCmd.PersistentFlags()
	nodeFlag = GenerateLicenseCmd.PersistentFlags()
	graceFlag = GenerateLicenseCmd.PersistentFlags()
	debugFlag = GenerateLicenseCmd.PersistentFlags()

	customerFlag.StringVar(&claims.Customer, "customer", "", "customer name")
	expFlag.StringVar(&exp, "expiry", "", "license expiration date in MM/DD/YYYY format. Expires on that day at 23:59:59 cluster local timezone.")
	nodeFlag.IntVar(&claims.Nodes, "nodes", 0, "number of nodes customer is licensed for. Set this to -1 for unlimited nodes license.")
	graceFlag.IntVar(&claims.GracePeriod, "graceperiod", 90, "number of days the cluster will keep working after the license expires")
	debugFlag.BoolVar(&debug, "debug", false, "print debug information about the license fields")

	GenerateLicenseCmd.MarkPersistentFlagRequired("customer")
	GenerateLicenseCmd.MarkPersistentFlagRequired("expiry")
	GenerateLicenseCmd.MarkPersistentFlagRequired("nodes")

	absPrivKeyPath, _ = filepath.Abs(privKeyPath)
	absCertPath, _ = filepath.Abs(certPath)
}

var GenerateLicenseCmd = &cobra.Command{
	Use: "generate license",
	Run: func(cmd *cobra.Command, args []string) {

		claims.Expiry = parseExpiryDate(exp)
		claims.LicenseID = uuid.NewV4().String()

		// We set CheckinInterval to 0 so it's an offline license since we don't have call-home server in v2.1.
		// This will be a flag when we have the licensing server, with default check-in interval set to a week.
		claims.CheckinInterval = time.Duration(0)

		// This might be configurable in future. Right now we just license all the features.
		claims.Features = []string{"cnx", "all"}

		// This might be used in future. Or it could be used for debugging.
		claims.IssuedAt = jwt.NewNumericDate(time.Now().UTC())

		if len(claims.Customer) < 3 {
			log.Fatal("[ERROR] Customer name must be at least 3 charecters long")
		}

		fmt.Println("Confirm the license information:")
		fmt.Printf("Customer name:                            %s\n", claims.Customer)
		fmt.Printf("Number of nodes ( -1 = unlimited nodes):  %d\n", claims.Nodes)
		fmt.Printf("License term expiration date:             %v\n", claims.Claims.Expiry.Time())
		fmt.Printf("Features:                                 %s\n", claims.Features)
		fmt.Printf("Checkin interval (0 = offline license):   %d\n", claims.CheckinInterval)
		fmt.Printf("Grace period (days):                      %d\n", claims.GracePeriod)
		fmt.Printf("License ID (auto-generated):              %s\n", claims.LicenseID)
		fmt.Println("Is the license information correct? [y/N]")

		var valid string
		fmt.Scanf("%s", &valid)

		if strings.ToLower(valid) != "y" {
			os.Exit(1)
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
