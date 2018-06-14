// Package testutils provides a test decode method which can be used with certs not signed by the official
// tigera root key pair. This is useful for testing license generation without needing to pass around the real certs.
// But it should not be used for anything else! Do not import this into production code under any circumstance!!
package testutils

import (
	"fmt"

	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/tigera/licensing/client"
	cryptolicensing "github.com/tigera/licensing/crypto"
	"gopkg.in/square/go-jose.v2/jwt"
)

// IsDevLicense is an error returned when using the dev license decoder. It is used to prevent
// clients from accidentally using the dev decoder.
type IsDevLicense struct {
	s string
}

func (e *IsDevLicense) Error() string {
	return e.s
}

// DecodeDevLicense takes a license resource and decodes the claims using the provided Root.
//
// DO NOT USE THIS IN ANY PRODUCTION CODE SHIPPED IN ANY BINARIES!!
//
// This Method is for testing use only. It returns the decoded client.LicenseClaims and an error.
// To prevent accidental use of this function, the error is _always_ non-nil. If the function succeeded,
// it will be an IsDevLicense error. Any other error means the license is corrupted.
func DecodeDevLicense(lic api.LicenseKey, devRoot string) (client.LicenseClaims, error) {
	tok, err := jwt.ParseSignedAndEncrypted(lic.Spec.Token)
	if err != nil {
		return client.LicenseClaims{}, fmt.Errorf("error parsing license: %s", err)
	}

	nested, err := tok.Decrypt([]byte("i༒2ஹ阳0?!pᄚ3-)0$߷५ૠm"))
	if err != nil {
		return client.LicenseClaims{}, fmt.Errorf("error decrypting license: %s", err)
	}

	cert, err := cryptolicensing.LoadCertFromPEM([]byte(lic.Spec.Certificate))
	if err != nil {
		return client.LicenseClaims{}, fmt.Errorf("error loading license certificate: %s", err)
	}

	rootCert, err := cryptolicensing.LoadCertFromPEM([]byte(devRoot))
	if err != nil {
		return client.LicenseClaims{}, fmt.Errorf("error loading license certificate: %s", err)
	}

	// Check if the certificate was signed by the provided root certificate.
	if err = cert.CheckSignatureFrom(rootCert); err != nil {
		return client.LicenseClaims{}, fmt.Errorf("failed to verify the certificate: %s", err)
	}

	var claims client.LicenseClaims
	if err := nested.Claims(cert.PublicKey, &claims); err != nil {
		return client.LicenseClaims{}, fmt.Errorf("error parsing license claims: %s", err)
	}

	// Return the succesfully decoded claim and an IsDevLicense error.
	return claims, &IsDevLicense{"This license is for development only!"}
}
