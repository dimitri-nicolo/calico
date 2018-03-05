package client

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/dgrijalva/jwt-go"
	cryptolicensing "github.com/tigera/licensing/crypto"
	jwtv2 "gopkg.in/square/go-jose.v2"
)

// TODO move these into types package
type LicenseClaims struct {
	ID       string   `json:"id"`
	Nodes    string   `json:"nodes" validate:"required"`
	Name     string   `json:"name" validate:"required"`
	Features []string `json:"features"`
	jwt.StandardClaims
}

type License struct {
	Claims string `json:"claims"`
	Cert   string `json:"cert"`
	//Signature []byte `json:"signature"`
}

func (l License) String() string {
	return fmt.Sprintf("%s.\n%s", l.Claims, l.Cert)
}

func DecodeAndVerify(lic License) {

	encObjectRcv, err := jwtv2.ParseEncrypted(lic.Claims)
	if err != nil {
		panic(err)
	}

	decrypted, err := encObjectRcv.Decrypt([]byte("meepster124235546567546788888457"))
	if err != nil {
		panic(err)
	}

	fmt.Printf("\n * decrypted %v\n", string(decrypted))

	// Parse the serialized, protected JWS object. An error would indicate that
	// the given input did not represent a valid message.
	object, err := jwtv2.ParseSigned(string(decrypted))
	if err != nil {
		panic(err)
	}

	cert, err := cryptolicensing.LoadCertFromPEM([]byte(lic.Cert))

	// verify cert chain here ////////////////////////////

	// Now we can verify the signature on the payload. An error here would
	// indicate the the message failed to verify, e.g. because the signature was
	// broken or the message was tampered with.
	output, err := object.Verify(cert.PublicKey)
	if err != nil {
		panic(err)
	}

	var recLic LicenseClaims

	err = json.Unmarshal(output, &recLic)
	if err != nil {
		log.Fatalf("failed to unmarshal license: %s\n", err)
	}

	fmt.Printf("\n\n final: %v\n", recLic)
}
