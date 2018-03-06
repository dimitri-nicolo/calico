package symmetric

import (
	"crypto/rand"
	"crypto/aes"
	"crypto/cipher"
	"io"
	"fmt"
)

var (
	// Carefully selected key. It has to be 32-bit long.
	symKey = []byte("Rob likes tea & kills chickens!!")

	// RandomGen is a crypto pseudo-random generator.
	RandomGen = rand.Reader
)


func EncryptMessage(plaintext []byte) ([]byte, error) {
	c, err := aes.NewCipher(symKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(RandomGen, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func DecryptMessage(cipherText []byte) ([]byte, error) {
	c, err := aes.NewCipher(symKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(cipherText) < nonceSize {
		return nil, fmt.Errorf("cipher text is too short")
	}

	nonce, ciphertext := cipherText[:nonceSize], cipherText[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}