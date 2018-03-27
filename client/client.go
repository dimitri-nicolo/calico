package client

import (
	"time"

	"gopkg.in/square/go-jose.v2/jwt"

	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	cryptolicensing "github.com/tigera/licensing/crypto"
	"fmt"
	"crypto/x509"
)

// TODO: replace this with the actual cert once it's available.
const rootPEM = `-----BEGIN CERTIFICATE-----
MIID2DCCAsCgAwIBAgIRAMasaTPup3Kvwz1Y3Gx62w4wDQYJKoZIhvcNAQELBQAw
dTELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNh
biBGcmFuY2lzY28xFDASBgNVBAoTC1RpZ2VyYSBJbmMuMSMwIQYDVQQDExpUaWdl
cmEgSW5jLiBDZXJ0IEF1dGhvcml0eTAeFw0xODAzMjYxNjMxNTdaFw0xOTAzMjYx
NjMxNTZaMHUxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRYwFAYD
VQQHEw1TYW4gRnJhbmNpc2NvMRQwEgYDVQQKEwtUaWdlcmEgSW5jLjEjMCEGA1UE
AxMaVGlnZXJhIEluYy4gQ2VydCBBdXRob3JpdHkwggEiMA0GCSqGSIb3DQEBAQUA
A4IBDwAwggEKAoIBAQCzNYU41SwTeAuOXc9zJwQKYjWNO6+peLwnzcoWTnBQWkEa
a+nqOs7J4uTGiMQdwHdwGMNPSQLUzjZa4AmxljqOIVcKmfwIUPjuAAgbP27fuFCI
c/W8BMTseutBOYCwo+ZRlklSMv294kU4UeiGGjj1ndT994xnv198iMlG+7s737OH
7fk9+I1JWSPFoFvKAtrpCcPXos/pSNbLr2Ojp77Jc6EX2gXj5F3qb4ppqG8Rtbe+
bAHgeyfMBUKj8G/wZel1T4m25HrB/b5uWxsxcPkzJ9SA9WOlC3FsfoWyR3XbmkLu
DRIm8dCXhE8xiXG+X0UkHj0qX4R+W+6qRb5jeSGRAgMBAAGjYzBhMA4GA1UdDwEB
/wQEAwIBhjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBQwgD9AwdAx2j8933UU
7uZYVOw2kDAfBgNVHSMEGDAWgBQwgD9AwdAx2j8933UU7uZYVOw2kDANBgkqhkiG
9w0BAQsFAAOCAQEAMX0rkwC1b2+uNZDXpVHBYQ3KmkcL2GFxMypLEnH5W+PHJgEi
2JKz4g83M1zzrEHO+0RTwMuUVOD5/Mrwn8AcQvG5AOjdS946AEfhoso1RH1wy5y+
+cj9T8fELoFj/pwWN1zaCmLEh1WYX0unUM1XGlQD51S7fYg1g/4Z/HzBy4mgUE1M
4D3zV7y6S7l1VUwK9daStUZU9HN/Wa9Q0QsnASMSh5aAmPsC5uAfZT0Q7guD/O1b
eLb7zca40mf4yNSlgKRB9OpyP/XpbpOvPsNpVfzO5IFMGyJSZ6rB9zLoAaevJ2sH
41uNeuDypKIVzJ6Uz/hBiPp9JratP7x9iAukpA==
-----END CERTIFICATE-----`

var (
	// Symmetric key to encrypt and decrypt the JWT.
	// It has to be 32-byte long UTF-8 string.
	symKey = []byte("i༒2ஹ阳0?!pᄚ3-)0$߷५ૠm")

	// LicenseKey is a singleton resource, and has to have the name "default".
	ResourceName = "default"

	opts x509.VerifyOptions

)

func init() {
	// First, create the set of root certificates. For this example we only
	// have one. It's also possible to omit this in order to use the
	// default root set of the current operating system.
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(rootPEM))
	if !ok {
		panic("failed to load root cert")
	}

	opts = x509.VerifyOptions{
		Roots:   roots,
	}
}
// LicenseClaims contains all the license control fields.
// This includes custom JWT fields and the default ones.
type LicenseClaims struct {
	// LicenseID is a unique UUID assigned for each customer license.
	LicenseID string `json:"license_id"`

	// Node count is not enforced in v2.1. If it’s not set then it means it’s unlimited nodes
	// (site license)
	Nodes *int `json:"nodes" validate:"required"`

	// Customer is the name of the customer, so we can use the same name for multiple
	// licenses for a customer, but they'd have different LicenseID.
	Customer string `json:"name" validate:"required"`

	// ClusterGUID is an optional field that can be filled out to limit the use of license only on
	// a cluster with this specific ClusterGUID. This can also be used when client sends
	// license call-home checkins. Not used for v2.1.
	ClusterGUID string `json:"cluster_guid"`

	// Version of the license claims. Could be useful in future if/when we add new fields in
	// the license. This is different from the LicenseKey APIVersion field.
	Version string `json:"version"`

	// Features field is for future use.
	// We will default this with `[ “cnx”, “all”]` for v2.1
	Features []string `json:"features"`

	// GracePeriod is how many days the cluster will keep working even after
	// the license expires. This defaults to 90 days.
	// Currently not enforced.
	GracePeriod int `json:"grace_period"`

	// CheckinInterval is how frequently we call home (in hours).
	// Not used for v2.1. Defaults to once a week. If it’s not set then it’s an offline license.
	CheckinInterval *int `json:"checkin_interval"`

	// Include the default JWT claims.
	// Built-in field `Expiry` is used to set the license expiration date.
	// Built-in IssuedAt is set to the time of license generation (UTC), not used in v2.1.
	// Precision is day, and expires end of the day (on customer local timezone).
	jwt.Claims
}

// Decode takes a license resource and decodes the claims
// It returns the decoded LicenseClaims and an error. A non-nil error means the license is corrupted.
func Decode(lic api.LicenseKey) (LicenseClaims, error) {
	tok, err := jwt.ParseSignedAndEncrypted(lic.Spec.Token)
	if err != nil {
		return LicenseClaims{}, fmt.Errorf("error parsing license: %s", err)
	}

	nested, err := tok.Decrypt(symKey)
	if err != nil {
		return LicenseClaims{}, fmt.Errorf("error decrypting license: %s", err)
	}

	cert, err := cryptolicensing.LoadCertFromPEM([]byte(lic.Spec.Certificate))
	if err != nil {
		return LicenseClaims{}, fmt.Errorf("error loading license certificate: %s", err)
	}

	if _, err := cert.Verify(opts); err != nil {
		return LicenseClaims{}, fmt.Errorf("failed to verify the certificate: %s", err)
	}

	var claims LicenseClaims
	if err := nested.Claims(cert.PublicKey, &claims); err != nil {
		return LicenseClaims{}, fmt.Errorf("error parsing license claims: %s", err)
	}

	return claims, nil
}

// IsValid checks if the license is expired.
func (c LicenseClaims) IsValid() bool {
	return c.Claims.Expiry.Time().After(time.Now().Local())
}