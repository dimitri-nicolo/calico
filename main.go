package main

import (
	"fmt"
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
	uuid "github.com/satori/go.uuid"
	jose "gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	yaml "github.com/projectcalico/go-yaml-wrapper"
	"github.com/tigera/licensing/client"
	cryptolicensing "github.com/tigera/licensing/crypto"
)

func main() {
	customerID := uuid.NewV4().String()
	numNodes := 42

	claims := client.LicenseClaims{
		ID:          customerID,
		Nodes:       numNodes,
		Name:        "MyFavCustomer99",
		Features:    []string{"everything", "for", "you"},
		GracePeriod: 90,
		Term: 365,
		Claims: jwt.Claims{
			NotBefore: jwt.NumericDate(time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix()),
			Issuer:    "Gunjan's office number 5",
		},
	}

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

	fmt.Printf("\n ** on the WIRE: %v\n", licX)

	writeYAML(licX)

	// client.DecodeAndVerify(licX)

//	spew.Dump(licX)

	licY := client.ReadFile("./license.yaml")

	cl, valid := client.DecodeAndVerify(licY)
	spew.Dump(cl)
	fmt.Println(valid)
}

func writeYAML(license client.License) error {
	output, err := yaml.Marshal(license)
	if err != nil {
		return err
	}
	fmt.Printf("%s", string(output))
	return nil
}
