package main

import (
	"log"
	"time"


	cryptolicensing "github.com/tigera/licensing/crypto"
)

func main() {
	// Generate Pub/Priv key pair.
	priv, err := cryptolicensing.GenerateKeyPair()
	if err != nil {
		log.Fatalf("error generating pub/priv key pair")
	}

	err = cryptolicensing.SavePrivateKeyAsPEM(priv, "test_evil_private_key.pem")
	if err != nil {
		log.Fatalf("error saving private key to file: %s", err)
	}

	// Generate x.509 certificate.
	now := time.Now()
	// Valid for one year from now.
	then := now.Add(60 * 60 * 24 * 365 * 1000 * 1000 * 1000 * 5)
	derBytes, err := cryptolicensing.Generatex509Cert(now, then, priv)
	if err != nil {
		log.Fatalf("error generating x.509 certificate: %s", err)
	}

	err = cryptolicensing.SaveCertToFile(derBytes, "test_evil_cert.cer")
	if err != nil {
		log.Fatalf("error saving cert to file: %s", err)
	}

	err = cryptolicensing.SaveCertAsPEM(derBytes, "test_evil_cert.pem")
	if err != nil {
		log.Fatalf("error saving cert to file: %s", err)
	}
}