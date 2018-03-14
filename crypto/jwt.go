package crypto

import (
	"fmt"

	jose "gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"

	"github.com/tigera/licensing/client"
	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

var (
	// Symmetric key to encrypt and decrypt the JWT.
	// Carefully selected key. It has to be 32-bit long.
	symKey = []byte("Rob likes tea & kills chickens!!")

	// Tigera private key location.
	pkeyPath = "./tigera.io_private_key.pem"

	// Tigera license signing certificate path.
	certPath = "./tigera.io_certificate.pem"

	jwtTyp = jose.ContentType("JWT")

	jwtContentType = jose.ContentType("JWT")
)

func GetLicenseFromClaims(claims client.LicenseClaims, pkeyPath, certPath string) (*api.LicenseKey, error) {

	enc, err := jose.NewEncrypter(
		jose.A128GCM,
		jose.Recipient{
			Algorithm: jose.A128GCMKW,
			Key:       symKey,
		},
		(&jose.EncrypterOptions{}).WithType(jwtTyp).WithContentType(jwtContentType))
	if err != nil {
		return nil, fmt.Errorf("error generating claims: %s", err)
	}

	priv, err := ReadPrivateKeyFromFile(pkeyPath)
	if err != nil {
		return nil, fmt.Errorf("error reading private key: %s\n", err)
	}

	// Instantiate a signer using RSASSA-PSS (SHA512) with the given private key.
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.PS512, Key: priv}, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating signer: %s", err)
	}

	raw, err := jwt.SignedAndEncrypted(signer, enc).Claims(claims).CompactSerialize()
	if err != nil {
		return nil, fmt.Errorf("error signing the JWT: %s", err)
	}

	licX := api.NewLicenseKey()
	licX.Name = client.ResourceName
	licX.Spec.Token = raw
	licX.Spec.Certificate = ReadCertPemFromFile(certPath)

	return licX, nil
}
