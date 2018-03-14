package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	jose "gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/davecgh/go-spew/spew"

	yaml "github.com/projectcalico/go-yaml-wrapper"
	"github.com/tigera/licensing/client"
	cryptolicensing "github.com/tigera/licensing/crypto"
)

var (
	licClaimes                              client.LicenseClaims
	nameFlag, termFlag, nodeFlag, graceFlag *pflag.FlagSet

	// Symmetric key to encrypt and decrypt the JWT.
	// Carefully selected key. It has to be 32-bit long.
	symKey = []byte("Rob likes tea & kills chickens!!")
)

func init() {
	nameflag := GenerateLicenseCmd.PersistentFlags()
	termFlag = GenerateLicenseCmd.PersistentFlags()
	nodeFlag = GenerateLicenseCmd.PersistentFlags()
	graceFlag = GenerateLicenseCmd.PersistentFlags()
	nameflag.StringVar(&licClaimes.Name, "name", "", "customer name")
	termFlag.IntVar(&licClaimes.Term, "term", 0, "license term ")
	nodeFlag.IntVar(&licClaimes.Nodes, "nodes", 0, "number of nodes customer is licensed for")
	graceFlag.IntVar(&licClaimes.GracePeriod, "graceperiod", 90, "number of nodes customer is licensed for")
}

var GenerateLicenseCmd = &cobra.Command{
	Use: "generate license",
	Run: func(cmd *cobra.Command, args []string) {


		claims := GetLicenseProperties(false)

		now := time.Now()
		exp := now.Add(time.Hour * 24 * time.Duration(claims.Term))
		claims.NotBefore = jwt.NewNumericDate(exp)

		enc, err := jose.NewEncrypter(
			jose.A128GCM,
			jose.Recipient{
				Algorithm: jose.A128GCMKW,
				Key:       symKey,
			},
			(&jose.EncrypterOptions{}).WithType("JWT").WithContentType("JWT"))
		if err != nil {
			panic(err)
		}

		priv, err := cryptolicensing.ReadPrivateKeyFromFile("./privateKey.pem")
		if err != nil {
			log.Panicf("error reading private key: %s\n", err)
		}

		// Instantiate a signer using RSASSA-PSS (SHA512) with the given private key.
		signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.PS512, Key: priv}, nil)
		if err != nil {
			panic(err)
		}

		raw, err := jwt.SignedAndEncrypted(signer, enc).Claims(claims).CompactSerialize()
		if err != nil {
			panic(err)
		}

		licX := client.License{Claims: raw, Cert: cryptolicensing.ReadCertPemFromFile("./tigera.io.pem")}

		writeYAML(licX)

		fmt.Println("*******")
		spew.Dump(claims)

		fmt.Println("Created license file 'license.yaml'")
	},
}

func writeYAML(license client.License) error {
	output, err := yaml.Marshal(license)
	if err != nil {
		return err
	}
	fmt.Printf("%s", string(output))

	f, err := os.Create("./license.yaml")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	_, err = f.Write(output)
	if err != nil {
		panic(err)
	}
	return nil
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
