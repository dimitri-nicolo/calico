package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"time"

	cryptolicensing "github.com/tigera/licensing/crypto"
	"github.com/tigera/licensing/crypto/asymmetric"
	"github.com/tigera/licensing/crypto/symmetric"
)

func main() {
	message := []byte("My name is G U N J A N 5")

	// Hash the message.
	hashed := sha256.Sum256(message)

	// Generate Pub/Priv key pair.
	priv, err := cryptolicensing.GenerateKeyPair()
	if err != nil {
		log.Fatalf("error generating pub/priv key pair")
	}

	pub := priv.PublicKey

	privPem := cryptolicensing.ExportRsaPrivateKeyAsPemStr(priv)
	pubPem, err := cryptolicensing.ExportRsaPublicKeyAsPemStr(&pub)
	if err != nil {
		log.Fatalf("error exporting public key: %s\n", err)
	}

	fmt.Printf("Priv:\n%s\nPub:\n%s\n", privPem, pubPem)

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

	// Sign the message with private key.
	signature, err := asymmetric.SignMessage(priv, hashed[:])
	if err != nil {
		log.Fatalf("error signing the message: %s", err)
	}

	fmt.Printf("Signature: %x\n", signature)

	// Verify the signed message with public key.
	err = asymmetric.VerifySignedMessage(&pub, hashed[:], signature)
	if err != nil {
		log.Fatalf("failed to verify signature: %s", err)
	}
	fmt.Printf("signature is verified\n")

	// Asymmetrically encrypt the message with public key.
	cipherText, err := asymmetric.EncryptMessage(&pub, message)
	if err != nil {
		log.Fatalf("error encrypting message: %s", err)
	}

	fmt.Printf("Asymmetric encryption:\n%s => %x\n", message, cipherText)

	// Asymmetrically decrypt the message with private key.
	plainText, err := asymmetric.DecryptMessage(priv, cipherText)
	if err != nil {
		log.Fatalf("error decrypting message: %s\n", err)
	}

	fmt.Printf("Asymmetric decryption:\n%x => %s\n", cipherText, plainText)

	symciphertext, err := symmetric.EncryptMessage(message)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Symmetric encryption:\n%s => %x\n", message, symciphertext)

	symplaintext, err := symmetric.DecryptMessage(symciphertext)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Symmetric decryption:\n%x => %s\n", symciphertext, symplaintext)

}
