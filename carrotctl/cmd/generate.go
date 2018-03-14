package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/square/go-jose.v2/jwt"
	"github.com/davecgh/go-spew/spew"
	"github.com/satori/go.uuid"

	"github.com/tigera/licensing/client"
	cryptolicensing "github.com/tigera/licensing/crypto"
)

var (
	licClaimes                              client.LicenseClaims
	nameFlag, debugFlag, termFlag, nodeFlag, graceFlag *pflag.FlagSet

	// Tigera private key location.
	pkeyPath = "./tigera.io_private_key.pem"

	// Tigera license signing certificate path.
	certPath = "./tigera.io_certificate.pem"

	debug = false
)

func init() {
	nameFlag = GenerateLicenseCmd.PersistentFlags()
	termFlag = GenerateLicenseCmd.PersistentFlags()
	nodeFlag = GenerateLicenseCmd.PersistentFlags()
	graceFlag = GenerateLicenseCmd.PersistentFlags()
	debugFlag = GenerateLicenseCmd.PersistentFlags()
	nameFlag.StringVar(&licClaimes.Name, "name", "", "customer name")
	termFlag.IntVar(&licClaimes.Term, "term", 0, "license term ")
	nodeFlag.IntVar(&licClaimes.Nodes, "nodes", 0, "number of nodes customer is licensed for")
	graceFlag.IntVar(&licClaimes.GracePeriod, "graceperiod", 90, "number of nodes customer is licensed for")
	debugFlag.BoolVar(&debug, "debug", false, "print debug information about the license fields" )
}

var GenerateLicenseCmd = &cobra.Command{
	Use: "generate license",
	Run: func(cmd *cobra.Command, args []string) {


		claims := GetLicenseProperties(false)

		now := time.Now().UTC()
		exp := now.Add(time.Hour * 24 * time.Duration(claims.Term))
		claims.NotBefore = jwt.NewNumericDate(exp)
		claims.CustomerID = uuid.NewV4().String()

		// This might be used in future. Or it could be used for debugging.
		claims.IssuedAt = jwt.NewNumericDate(time.Now().UTC())

		lic, err := cryptolicensing.GetLicenseFromClaims(claims, pkeyPath, certPath)
		if err != nil{
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

	if !termFlag.Changed("term") || override {
		fmt.Println("Enter the license term (in days):")
		_, err := fmt.Scanf("%d", &lic.Term)
		if err != nil {
			fmt.Printf("[ERROR] invalid input: %s\n", err)
			os.Exit(1)
		}
	} else {
		lic.Term = licClaimes.Term
	}

	lic.GracePeriod = licClaimes.GracePeriod
	if override {
		fmt.Println("Enter the grace period (in days) [default 90]:")
		n, _ := fmt.Scanf("%d", &lic.GracePeriod)
		if n == 0 {
			lic.GracePeriod = licClaimes.GracePeriod
		}
	}

	fmt.Println("Confirm the license information:")
	fmt.Printf("Customer name:        %s\n", lic.Name)
	fmt.Printf("Number of nodes:      %d\n", lic.Nodes)
	fmt.Printf("License term (days):  %d\n", lic.Term)
	fmt.Printf("Grace period (days):  %d\n", lic.GracePeriod)
	fmt.Println("Is the license information correct? [y/N]")

	var valid string
	fmt.Scanf("%s", &valid)

	if strings.ToLower(valid) != "y" {
		return GetLicenseProperties(true)
	}

	return lic
}
