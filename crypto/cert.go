package crypto

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"math/rand"
	"time"

	"encoding/pem"
	"os"
)

const (
	randSerialSeed = 9223372036854775807

	CertCommonName = "tigera.io"

	CertOrgName = "Tigera Inc."

	CertTigeraDomain = "tigera.io"

	CertLicensingDomain = "licensing.tigera.io"

	CertEmailAddress = "licensing@tigera.io"

	CertType = "CERTIFICATE"
)

// Generatex509Cert generates an x.509 certificate with start and expiration time
// provided and the RSA private key provided and returns cert DER bytes and error.
// This function also populates Tigera org default information.
func Generatex509Cert(start, exp time.Time, priv *rsa.PrivateKey) ([]byte, error) {
	template := x509.Certificate{
		SerialNumber: big.NewInt(rand.Int63n(randSerialSeed)),
		Subject: pkix.Name{
			CommonName:   CertCommonName,
			Organization: []string{CertOrgName},
		},

		NotBefore: start,
		NotAfter:  exp,

		SubjectKeyId: []byte{1, 2, 3, 4},
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,

		BasicConstraintsValid: true,
		IsCA:           true,
		EmailAddresses: []string{CertEmailAddress},
		DNSNames:       []string{CertTigeraDomain, CertLicensingDomain},
	}

	return x509.CreateCertificate(RandomGen, &template, &template, &priv.PublicKey, priv)
}

func SaveCertToFile(derBytes []byte, filePath string) error {
	certCerFile, err := os.Create(filePath)
	if err != nil {
		return err
	}

	defer certCerFile.Close()

	certCerFile.Write(derBytes)

	return nil
}

func SaveCertAsPEM(derBytes []byte, filePath string) error {
	certPEMFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer certPEMFile.Close()

	pem.Encode(certPEMFile, &pem.Block{Type: CertType, Bytes: derBytes})

	return nil
}
