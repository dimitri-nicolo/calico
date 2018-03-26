package client

import (
	"time"

	"gopkg.in/square/go-jose.v2/jwt"

	"fmt"

	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	cryptolicensing "github.com/tigera/licensing/crypto"
)

var (
	// Symmetric key to encrypt and decrypt the JWT.
	// It has to be 32-byte long UTF-8 string.
	symKey = []byte("i༒2ஹ阳0?!pᄚ3-)0$߷५ૠm")

	// LicenseKey is a singleton resource, and has to have the name "default".
	ResourceName = "default"
)

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
