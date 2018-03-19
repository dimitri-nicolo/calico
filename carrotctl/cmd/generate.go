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
	licClaimes                                        client.LicenseClaims
	nameFlag, debugFlag, expFlag, nodeFlag, graceFlag *pflag.FlagSet

	// Tigera private key location.
	pkeyPath = "./tigera.io_private_key.pem"

	// Tigera license signing certificate path.
	certPath = "./tigera.io_certificate.pem"

	absPkeyPath, absCertPath string

	debug = false

	exp string
)

func init() {
	nameFlag = GenerateLicenseCmd.PersistentFlags()
	expFlag = GenerateLicenseCmd.PersistentFlags()
	nodeFlag = GenerateLicenseCmd.PersistentFlags()
	graceFlag = GenerateLicenseCmd.PersistentFlags()
	debugFlag = GenerateLicenseCmd.PersistentFlags()

	nameFlag.StringVar(&licClaimes.Name, "name", "", "customer name")
	expFlag.StringVar(&exp, "term", "", "license expiration date in DD/MM/YYYY format. Expires on that day at 23:59:59:999999999 (nanoseconds).")
	nodeFlag.IntVar(&licClaimes.Nodes, "nodes", 0, "number of nodes customer is licensed for")
	graceFlag.IntVar(&licClaimes.GracePeriod, "graceperiod", 90, "number of nodes customer is licensed for")
	debugFlag.BoolVar(&debug, "debug", false, "print debug information about the license fields")

	absPkeyPath, _ = filepath.Abs(pkeyPath)
	absCertPath, _ = filepath.Abs(certPath)
}

var GenerateLicenseCmd = &cobra.Command{
	Use: "generate license",
	Run: func(cmd *cobra.Command, args []string) {

		claims := GetLicenseProperties(false)

		claims.CustomerID = uuid.NewV4().String()

		// We set this to offline = true since we don't have call-home server in v2.1.
		// This will be a flag when we have the licensing server.
		claims.Offline = true

		// This might be used in future. Or it could be used for debugging.
		claims.IssuedAt = jwt.NewNumericDate(time.Now().UTC())

		lic, err := client.GetLicenseFromClaims(claims, absPkeyPath, absCertPath)
		if err != nil {
			log.Fatalf("error generating license from claims: %s", err)
		}

		err = WriteYAML(*lic, claims.Name)
		if err != nil {
			log.Fatalf("error creating the license file: %s", err)
		}

		if debug {
			spew.Dump(claims)
		}
	},
}

func GetLicenseProperties(override bool) client.LicenseClaims {
	var lic client.LicenseClaims

	if licClaimes.Name == "" || override {
		fmt.Println("Enter the customer name:")
		n, err := fmt.Scanf("%s", &lic.Name)
		if n == 0 {
			fmt.Println("[ERROR] Company name cannot be empty!")
			return GetLicenseProperties(true)
		}
		if err != nil {
			fmt.Printf("[ERROR] invalid input: %s\n", err)
			os.Exit(1)
		}
	} else {
		lic.Name = licClaimes.Name
	}

	if !nodeFlag.Changed("nodes") || override {
		fmt.Println("Enter number of nodes the customer is licensed for:")
		_, err := fmt.Scanf("%d", &lic.Nodes)
		if err != nil {
			fmt.Printf("[ERROR] invalid input: %s\n", err)
			os.Exit(1)
		}
	} else {
		lic.Nodes = licClaimes.Nodes
	}

	if !expFlag.Changed("term") || override {
		fmt.Println("Enter the license expiration date (MM/DD/YYYY):")
		var licExpStr string
		_, err := fmt.Scanf("%s", &licExpStr)
		if err != nil {
			fmt.Printf("[ERROR] invalid input: %s\n", err)
			os.Exit(1)
		}

		lic.Claims.NotBefore = parseDate(licExpStr)

	} else {
		lic.Claims.NotBefore = parseDate(exp)
	}

	lic.GracePeriod = licClaimes.GracePeriod
	if override {
		fmt.Println("Enter the grace period (in days) [default 90]:")
		n, _ := fmt.Scanf("%d", &lic.GracePeriod)
		if n == 0 {
			lic.GracePeriod = 90
		}
	}

	fmt.Println("Confirm the license information:")
	fmt.Printf("Customer name:        %s\n", lic.Name)
	fmt.Printf("Number of nodes:      %d\n", lic.Nodes)
	fmt.Printf("License term expiration date:  %v\n", lic.Claims.NotBefore.Time())
	fmt.Printf("Grace period (days):  %d\n", lic.GracePeriod)
	fmt.Println("Is the license information correct? [y/N]")

	var valid string
	fmt.Scanf("%s", &valid)

	if strings.ToLower(valid) != "y" {
		return GetLicenseProperties(true)
	}

	return lic
}

func parseDate(dateStr string) jwt.NumericDate {
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

	return jwt.NewNumericDate(time.Date(yyyy, time.Month(mm), dd, 23, 59, 59, 59, time.Local))
}
