package utils

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	uuid "github.com/satori/go.uuid"
	jose "gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	yaml "github.com/projectcalico/go-yaml-wrapper"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"github.com/tigera/licensing/client"
	cryptolicensing "github.com/tigera/licensing/crypto"
)

func e2eFlow() {
	customerID := uuid.NewV4().String()
	numNodes := 42

	claims := client.LicenseClaims{
		LicenseID:   customerID,
		Nodes:       &numNodes,
		Customer:    "MyFavCustomer99",
		Features:    []string{"everything", "for", "you"},
		GracePeriod: 90,
		Claims: jwt.Claims{
			NotBefore: jwt.NumericDate(time.Date(2019, 10, 10, 12, 0, 0, 0, time.UTC).Unix()),
			Issuer:    "Gunjan's office number 5",
		},
	}

	enc, err := jose.NewEncrypter(
		jose.A128GCM,
		jose.Recipient{
			Algorithm: jose.A128GCMKW,
			Key:       []byte("Rob likes tea & kills chickens!!"),
		},
		(&jose.EncrypterOptions{}).WithType("JWT").WithContentType("JWT"))
	if err != nil {
		panic(err)
	}

	priv, err := cryptolicensing.ReadPrivateKeyFromFile("./tigera.io_private_key.pem")
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

	licX := api.NewLicenseKey()
	licX.Name = client.ResourceName
	licX.Spec.Token = raw
	licX.Spec.Certificate = cryptolicensing.ReadCertPemFromFile("./tigera.io_certificate.pem")

	fmt.Printf("\n ** on the WIRE: %v\n", licX)

	writeYAML(*licX)

	licY := ReadFile("./license.yaml")

	cl, valid := client.Decode(licY)
	spew.Dump(cl)
	fmt.Println(valid)
}

func writeYAML(license api.LicenseKey) error {
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
