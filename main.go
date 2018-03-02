package main

import (
	"fmt"
	"log"
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"crypto/sha256"

	cryptolicensing "github.com/tigera/licensing/crypto"
	"github.com/tigera/licensing/crypto/asymmetric"
	//"github.com/tigera/licensing/crypto/symmetric"

	"encoding/json"

	"github.com/tigera/licensing/client"
	"github.com/tigera/licensing/crypto/symmetric"
	"bytes"
)

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

func main() {

	//	message := []byte("My name is G U N J A N 5")
	//	message := []byte("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lMDAwMCIsImFkbWluIjp0cnVlfQ.GeYDu1EGbeLldjwiUqM3PAqdP_WEq-xmEnL6d7hDt7k")

	// Hash the message.
	//hashed := sha256.Sum256(message)

	// Generate Pub/Priv key pair.
	priv, err := cryptolicensing.GenerateKeyPair()
	if err != nil {
		log.Fatalf("error generating pub/priv key pair")
	}

	//pub := priv.PublicKey

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
	// certPem := cryptolicensing.ExportCertAsPemStr(derBytes)

	customerID := "21124-345235-3464574e574-455235" //uuid.NewV4()
	numNodes := "42"

	claims := LicenseClaims{
		ID:       customerID,
		Nodes:    numNodes,
		Name:     "MyFavCustomer99",
		Features: []string{"everything", "for", "you"},
		StandardClaims: jwt.StandardClaims{
			NotBefore: time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
			Issuer:    "Gunjan's office number 5",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte("meepster"))

	fmt.Println(tokenString, err)

	//fmt.Println(token.Method.Alg())

	//hashedJWT := make([]byte, 1024)
	//
	//base64.StdEncoding.Encode(hashedJWT, []byte(tokenString))

	lic := License{
		Claims: tokenString,
		Cert:   cryptolicensing.ExportCertAsPemStr(derBytes),
	}

	// fmt.Printf("^^^ lic string: %v\n", lic)
	hashed := sha256.Sum256([]byte(lic.String()))

	// Sign the message with private key.
	signature, err := asymmetric.SignMessage(priv, hashed[:])
	if err != nil {
		log.Fatalf("error signing the message: %s", err)
	}

	lic.Signature = signature

	b, err := json.Marshal(lic)
	if err != nil {
		log.Fatalf("failed to marshal license: %s\n", err)
	}

	symciphertext, err := symmetric.EncryptMessage(b)
	if err != nil {
		log.Fatal(err)
	}

	symciphertext = bytes.Trim(symciphertext, "\x00")
	// fmt.Printf("Signature: %x\n", signature)
	fmt.Println(symciphertext)
	fmt.Println(" ---------------------------------------------------------------------")
	//fmt.Printf("** WIRE: Symmetric encryption:\n%s ----------> %x\n", tokenString, symciphertext)
	copy(symciphertext[:], symciphertext[2:])
	fmt.Println(" ---------------------------------------------------------------------")
	fmt.Println(symciphertext)
	// ---------------------------------------------------------------------
	// ---------------------------------------------------------------------
	//

	client.DecodeAndVerify(symciphertext)

	//symplaintext, err := symmetric.DecryptMessage(symciphertext)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//var recLic License
	//
	//err = json.Unmarshal(symplaintext, &recLic)
	//if err != nil {
	//	log.Fatalf("failed to unmarshal license: %s\n", err)
	//}
	////fmt.Printf("Symmetric decryption:\n%x ------> %s\n", symciphertext, symplaintext)
	//
	//cert, err := cryptolicensing.LoadCertFromPEM([]byte(recLic.Cert))
	//
	//// verify cert chain here ////////////////////////////
	//
	//// Verify the signed message with public key.
	//err = asymmetric.VerifySignedMessage(cert.PublicKey.(*rsa.PublicKey), hashed[:], recLic.Signature)
	//if err != nil {
	//	log.Fatalf("failed to verify signature: %s", err)
	//}
	//fmt.Printf("signature is verified\n")
	//
	//tokenRcv, err := jwt.Parse(string(recLic.Claims), func(token *jwt.Token) (interface{}, error) {
	//	return []byte("meepster"), nil
	//})
	//
	//if tokenRcv.Valid {
	//	log.Println("Token is valid!")
	//} else if ve, ok := err.(*jwt.ValidationError); ok {
	//	if ve.Errors&jwt.ValidationErrorMalformed != 0 {
	//		log.Fatalf("Not a valid JWT token: %s", symplaintext)
	//	} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
	//		log.Println("JWT token is either expired or not active yet")
	//	} else {
	//		log.Fatalf("couldn't handle this token: %s\n", err)
	//	}
	//} else {
	//	log.Fatalf("couldn't handle this token: %s\n", err)
	//}
	//
	//fmt.Printf("****** rcv: %v\n\n", tokenRcv.Claims)

	//token.Method.Verify("")

	//fmt.Println(token)

	//privPem := cryptolicensing.ExportRsaPrivateKeyAsPemStr(priv)
	//pubPem, err := cryptolicensing.ExportRsaPublicKeyAsPemStr(&pub)
	//if err != nil {
	//	log.Fatalf("error exporting public key: %s\n", err)
	//}

	//fmt.Printf("Priv:\n%s\nPub:\n%s\n", privPem, pubPem)

	//// Asymmetrically encrypt the message with public key.
	//cipherText, err := asymmetric.EncryptMessage(&pub, message)
	//if err != nil {
	//	log.Fatalf("error encrypting message: %s", err)
	//}
	//
	//fmt.Printf("Asymmetric encryption:\n%s ------> %x\n", message, cipherText)
	//
	//// Asymmetrically decrypt the message with private key.
	//plainText, err := asymmetric.DecryptMessage(priv, cipherText)
	//if err != nil {
	//	log.Fatalf("error decrypting message: %s\n", err)
	//}
	//
	//fmt.Printf("Asymmetric decryption:\n%x ------> %s\n", cipherText, plainText)

}
