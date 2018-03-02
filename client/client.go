package client

import (
	"github.com/tigera/licensing/crypto/symmetric"
	"log"
	"encoding/json"
	"github.com/tigera/licensing/crypto/asymmetric"
	cryptolicensing "github.com/tigera/licensing/crypto"
	"crypto/rsa"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"crypto/sha256"
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
	Claims    string `json:"claims"`
	Cert      string `json:"cert"`
	Signature []byte `json:"signature"`
}

func (l License) String() string {
	return fmt.Sprintf("%s.\n%s", l.Claims, l.Cert)
}

func DecodeAndVerify(symciphertext []byte) {


	symplaintext, err := symmetric.DecryptMessage(symciphertext)
	if err != nil {
		log.Fatal(err)
	}

	var recLic License

	err = json.Unmarshal(symplaintext, &recLic)
	if err != nil {
		log.Fatalf("failed to unmarshal license: %s\n", err)
	}
	//fmt.Printf("Symmetric decryption:\n%x ------> %s\n", symciphertext, symplaintext)

	cert, err := cryptolicensing.LoadCertFromPEM([]byte(recLic.Cert))

	// verify cert chain here ////////////////////////////

	hashed := sha256.Sum256([]byte(recLic.String()))

	// Verify the signed message with public key.
	err = asymmetric.VerifySignedMessage(cert.PublicKey.(*rsa.PublicKey), hashed[:], recLic.Signature)
	if err != nil {
		log.Fatalf("failed to verify signature: %s", err)
	}
	fmt.Printf("signature is verified\n")

	tokenRcv, err := jwt.Parse(string(recLic.Claims), func(token *jwt.Token) (interface{}, error) {
		return []byte("meepster"), nil
	})

	if tokenRcv.Valid {
		log.Println("Token is valid!")
	} else if ve, ok := err.(*jwt.ValidationError); ok {
		if ve.Errors&jwt.ValidationErrorMalformed != 0 {
			log.Fatalf("Not a valid JWT token: %s", symplaintext)
		} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
			log.Println("JWT token is either expired or not active yet")
		} else {
			log.Fatalf("couldn't handle this token: %s\n", err)
		}
	} else {
		log.Fatalf("couldn't handle this token: %s\n", err)
	}

	fmt.Printf("****** rcv: %v\n\n", tokenRcv.Claims)
}
