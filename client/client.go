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
MIID1zCCAr+gAwIBAgIQLq4dtj0XmIrPqYb8CN7yoTANBgkqhkiG9w0BAQsFADB1
MQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMNU2Fu
IEZyYW5jaXNjbzEUMBIGA1UEChMLVGlnZXJhIEluYy4xIzAhBgNVBAMTGlRpZ2Vy
YSBJbmMuIENlcnQgQXV0aG9yaXR5MB4XDTE4MDMyNjE2NDMyOVoXDTE5MDMyNjE2
NDMyOFowdTELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNV
BAcTDVNhbiBGcmFuY2lzY28xFDASBgNVBAoTC1RpZ2VyYSBJbmMuMSMwIQYDVQQD
ExpUaWdlcmEgSW5jLiBDZXJ0IEF1dGhvcml0eTCCASIwDQYJKoZIhvcNAQEBBQAD
ggEPADCCAQoCggEBAL/DjPf5w2obb3fItwIf9Jt3ovTQfD08ZiHUSluD/+mvOOK5
wnyzTsS/AXWa67v3W5iY1fe9+fT0r5myfRfmuAI0tmWu6Q/PUJufAkAC6PKMRxa+
yCErQIjpLkQL4sSTq8l0yNg7yBXBDBgqBDxevMzIMMBbgnyrvhCG3LLlgaeNFtDL
/BdandVcX5dcioixv/R+qx3KMIZEKaW++tDFFQAGpuXEhw0ztTykStnvImZq34eV
Qcj2qTs7GXXoNTNt6CwyXzCh9Gx8+db7Zb/AuMyGdQNJ7RRSjIILJp6Y/cGmX2NC
OOTyCw42L+ARKPKBowcDNLEIw2cBwY47G1zN9KUCAwEAAaNjMGEwDgYDVR0PAQH/
BAQDAgGGMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFGib49CAau909ncUpB/a
akasszreMB8GA1UdIwQYMBaAFGib49CAau909ncUpB/aakasszreMA0GCSqGSIb3
DQEBCwUAA4IBAQBIzRPohd5AtllbAnWCVttMRH7Y97GSFpPI17J2mtqS9XhrQUT/
TUSbF542W1k9WXhOWJWzZi01/P+Hp+t+qy/iz43yWxXvj4/SIpL34gX7DKWkTaQD
N+bc0jx5v3wb44jQpyAanFdX/upswnp26QO03IjitgHa5mAqNcZfTCv0RSCQJSaK
91S1lpS35aZExZ1AL6atfdyQ+mkUMOoJpa7NwSraLEOvO39AJ4asP513+mZ54+wC
kojBo4VD7iOZCgMrs1IUC+1AXz7WiueZcTOHVsFjE8MmKNsrVAnHH3B3lutxCIWU
Nj31BRTwnoxsuc6T4FJ862XNBLoEQRjOzJa3
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