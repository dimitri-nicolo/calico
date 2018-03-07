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
		// Generate Pub/Priv key pair.
		priv, err := cryptolicensing.GenerateKeyPair()
		if err != nil {
			log.Fatalf("error generating pub/priv key pair")
		}

		err = cryptolicensing.SavePrivateKeyAsPEM(priv, "privateKey.pem")
		if err != nil {
			log.Fatalf("error saving private key to file: %s", err)
		}

		// Generate x.509 certificate.
		now := time.Now()
		// Valid for one year from now.
		then := now.Add(60 * 60 * 24 * 365 * 1000 * 1000 * 1000)
		derBytes, err := cryptolicensing.Generatex509Cert(now, then, priv)
		if err != nil {
			log.Fatalf("error generating x.509 certificate: %s", err)
		}

		err = cryptolicensing.SaveCertToFile(derBytes, "tigera.io.cer")
		if err != nil {
			log.Fatalf("error saving cert to file: %s", err)
		}

		err = cryptolicensing.SaveCertAsPEM(derBytes, "tigera.io.pem")
		if err != nil {
			log.Fatalf("error saving cert to file: %s", err)
		}

		claims := GetLicenseProperties(false)

		enc, err := jose.NewEncrypter(
			jose.A128GCM,
			jose.Recipient{
				Algorithm: jose.A128GCMKW,
				Key:       []byte("meepster124235546567546788888457"),
			},
			(&jose.EncrypterOptions{}).WithType("JWT").WithContentType("JWT"))
		if err != nil {
			panic(err)
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

		licX := client.License{Claims: raw, Cert: cryptolicensing.ExportCertAsPemStr(derBytes)}

		writeYAML(licX)

		fmt.Println("*******")
		spew.Dump(claims)
	},
}

func writeYAML(license client.License) error {
	output, err := yaml.Marshal(license)
	if err != nil {
		return err
	}
	fmt.Printf("%s", string(output))
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
