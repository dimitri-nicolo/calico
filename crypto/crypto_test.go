package crypto_test

import (
	"testing"
	"log"
	"time"

cryptolicensing "github.com/tigera/licensing/crypto"
)

func TestCryptoBasics(t *testing.T) {
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
}