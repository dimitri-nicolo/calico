package asymmetric

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
)

const (
	encryptionLable = "shhh_tigera"
)

var (
	DefaultSigningHash = crypto.SHA256

	// RandomGen is a crypto pseudo-random generator.
	RandomGen = rand.Reader

	DefaultEncryptionHash = sha256.New()
)

func SignMessage(priv *rsa.PrivateKey, message []byte) ([]byte, error) {
	signature, err := rsa.SignPKCS1v15(RandomGen, priv, DefaultSigningHash, message[:])
	if err != nil {
		return nil, fmt.Errorf("error signing the message: %s", err)
	}

	return signature, nil
}

func VerifySignedMessage(pub *rsa.PublicKey, message, signature []byte) error {
	err := rsa.VerifyPKCS1v15(pub, DefaultSigningHash, message[:], signature)
	if err != nil {
		return fmt.Errorf("signature not verified on the message: %s", err)
	}

	return nil
}

func EncryptMessage(pub *rsa.PublicKey, message []byte) ([]byte, error) {
	cipher, err := rsa.EncryptOAEP(DefaultEncryptionHash, RandomGen, pub, message, []byte(encryptionLable))
	if err != nil {
		return nil, fmt.Errorf("error encrypting message")
	}

	return cipher, nil
}

func DecryptMessage(priv *rsa.PrivateKey, cipher []byte) ([]byte, error) {
	plainText, err := rsa.DecryptOAEP(DefaultEncryptionHash, RandomGen, priv, cipher, []byte(encryptionLable))
	if err != nil {
		return nil, fmt.Errorf("error decrypting message: %s", err)
	}

	return plainText, nil
}
