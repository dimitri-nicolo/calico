package client

import (
	"fmt"

	"gopkg.in/square/go-jose.v2/jwt"

	cryptolicensing "github.com/tigera/licensing/crypto"
)

// TODO move these into types package
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

func DecodeAndVerify(lic License) {
	tok, err := jwt.ParseSignedAndEncrypted(lic.Claims)
	if err != nil {
		panic(err)
	}

	nested, err := tok.Decrypt([]byte("meepster124235546567546788888457"))
	if err != nil {
		panic(err)
	}

	cert, err := cryptolicensing.LoadCertFromPEM([]byte(lic.Cert))
	if err != nil {
		panic(err)
	}

	var out LicenseClaims
	if err := nested.Claims(cert.PublicKey, &out); err != nil {
		panic(err)
	}

	fmt.Printf("*** %v\n", out)
}
