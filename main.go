package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	uuid "github.com/satori/go.uuid"

	jose "gopkg.in/square/go-jose.v2"

	"github.com/tigera/licensing/client"
	cryptolicensing "github.com/tigera/licensing/crypto"
)

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

//func (l License) String() string {
//	return fmt.Sprintf("%s.\n%s", l.Claims, l.Cert)
//}

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

	customerID := uuid.NewV4().String()
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

	// Instantiate a signer using RSASSA-PSS (SHA512) with the given private key.
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.PS512, Key: priv}, nil)
	if err != nil {
		panic(err)
	}

	b2, err := json.Marshal(claims)
	if err != nil {
		log.Fatalf("failed to marshal license: %s\n", err)
	}

	// Sign a sample payload. Calling the signer returns a protected JWS object,
	// which can then be serialized for output afterwards. An error would
	// indicate a problem in an underlying cryptographic primitive.
	//var payload = []byte("Lorem ipsum dolor sit amet")
	object, err := signer.Sign(b2)
	if err != nil {
		panic(err)
	}

	// Serialize the encrypted object using the full serialization format.
	// Alternatively you can also use the compact format here by calling
	// object.CompactSerialize() instead.
	serialized := object.FullSerialize()

	fmt.Printf("serialized: %s\n", serialized)

	// publicKey := &privateKey.PublicKey
	encrypter, err := jose.NewEncrypter(jose.A128GCM, jose.Recipient{Algorithm: jose.A128GCMKW, Key: []byte("meepster124235546567546788888457")}, nil)
	if err != nil {
		panic(err)
	}

	encObject, err := encrypter.Encrypt([]byte(serialized))
	if err != nil {
		panic(err)
	}

	encSerialized := encObject.FullSerialize()

	licX := client.License{Claims: encSerialized, Cert: cryptolicensing.ExportCertAsPemStr(derBytes)}

	// -----------------------------------------------------------------

	fmt.Printf("\n ** on the WIRE: %v\n", licX)

	client.DecodeAndVerify(licX)

}
