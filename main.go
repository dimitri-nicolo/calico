package main

import (
	"fmt"
	"log"
	"time"

	uuid "github.com/satori/go.uuid"
	jose "gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/tigera/licensing/client"
	cryptolicensing "github.com/tigera/licensing/crypto"
)

type LicenseClaims struct {
	ID       string   `json:"id"`
	Nodes    string   `json:"nodes" validate:"required"`
	Name     string   `json:"name" validate:"required"`
	Features []string `json:"features"`
	jwt.Claims
}

type License struct {
	Claims string `json:"claims"`
	Cert   string `json:"cert"`
}

func main() {
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

	customerID := uuid.NewV4().String()
	numNodes := "42"

	claims := LicenseClaims{
		ID:       customerID,
		Nodes:    numNodes,
		Name:     "MyFavCustomer99",
		Features: []string{"everything", "for", "you"},
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

	fmt.Printf("\n ** on the WIRE: %v\n", licX)

	client.DecodeAndVerify(licX)
}
