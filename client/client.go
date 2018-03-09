package client

import (
	"fmt"
	"io/ioutil"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2/jwt"

	yaml "github.com/projectcalico/go-yaml-wrapper"
	cryptolicensing "github.com/tigera/licensing/crypto"
)

var (
	// Symmetric key to encrypt and decrypt the JWT.
	// Carefully selected key. It has to be 32-bit long.
	symKey = []byte("Rob likes tea & kills chickens!!")
)

// TODO move these into types package
// LicenseClaims contains all the license control fields.
// This includes custom JWT fields and the default ones.
type LicenseClaims struct {
	ID          string   `json:"id"`
	Nodes       int      `json:"nodes" validate:"required"`
	Name        string   `json:"name" validate:"required"`
	Features    []string `json:"features"`
	GracePeriod int      `json:"grace_period"`
	Term        int      `json:"term"`
	jwt.Claims
}

// License contains signed and encrypted JWT (aka JWS & JWE - nested)
// as well as the certificate signed by Tigera root certificate to
// verify if the license was issued by Tigera.
type License struct {
	Claims string `json:"claims" yaml:"Claims"`
	Cert   string `json:"cert" yaml:"Cert"`
}

// DecodeAndVerify takes a license resource and will verify and decode the claims
// It returns the decoded LicenseClaims and a bool indicating if the license is valid.
func DecodeAndVerify(lic License) (LicenseClaims, bool) {
	tok, err := jwt.ParseSignedAndEncrypted(lic.Claims)
	if err != nil {
		log.Errorf("error parsing license: %s", err)
		return LicenseClaims{}, false
	}

	nested, err := tok.Decrypt(symKey)
	if err != nil {
		log.Errorf("error decrypting license: %s", err)
		return LicenseClaims{}, false
	}

	cert, err := cryptolicensing.LoadCertFromPEM([]byte(lic.Cert))
	if err != nil {
		log.Errorf("error loading license certificate: %s", err)
		return LicenseClaims{}, false
	}

	var out LicenseClaims
	if err := nested.Claims(cert.PublicKey, &out); err != nil {
		log.Errorf("error parsing license claims: %s", err)
		return LicenseClaims{}, false
	}

	fmt.Printf("*** %v\n", out)

	return out, out.Claims.NotBefore.Time().After(time.Now())
}

// This is temp, until libcalico-go resource is merged.
func ReadFile(path string) License {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	var lic License
	err = yaml.Unmarshal(data, &lic)
	if err != nil {
		panic(err)
	}

	return lic
}
