package client

import (
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2/jwt"

	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	cryptolicensing "github.com/tigera/licensing/crypto"
)

var (
	// Symmetric key to encrypt and decrypt the JWT.
	// Carefully selected key. It has to be 32-bit long.
	symKey = []byte("Rob likes tea & kills chickens!!")

	// LicenseKey is a singleton resource, and has to have the name "default".
	ResourceName = "default"
)

// LicenseClaims contains all the license control fields.
// This includes custom JWT fields and the default ones.
type LicenseClaims struct {
	// LicenseID is a unique UUID assigned for each customer license.
	LicenseID          string   `json:"license_id"`

	// Node count is not enforced in v2.1. Set to -1 for unlimited nodes (site license)
	Nodes       int      `json:"nodes" validate:"required"`

	// Customer is the name of the customer, so we can use the same name for multiple
	// licenses for a customer, but they'd have different LicenseID.
	Customer        string   `json:"name" validate:"required"`

	// ClusterGUID is an optional field that can be filled out to limit the use of license only on
	// a cluster with this specific ClusterGUID. This can also be used when client sends
	// license call-home checkins. Not used for v2.1.
	ClusterGUID string `json:"cluster_guid"`

	// Version of the license claims. Could be useful in future if/when we add new fields in
	// the license. This is different from the LicenseKey APIVersion field.
	Version string `json:"version"`

	// Features field is for future use.
	// We will default this with `[ “cnx”, “all”]` for v2.1
	Features    []string `json:"features"`

	// GracePeriod is how many days the cluster will keep working even after
	// the license expires. This defaults to 90 days.
	// Currently not enforced.
	GracePeriod int      `json:"grace_period"`

	// CheckinInterval is how frequently we call home.
	// Not used for v2.1. Defaults to once a week. Set to 0 for offline license.
	CheckinInterval time.Duration `json:"checkin_interval"`

	// Include the default JWT claims.
	// Built-in field `Expiry` is used to set the license expiration date.
	// Precision is day, and expires at 23:59:59:999999999 (down to nanoseconds) on that
	// date (on customer local timezone).
	jwt.Claims
}

// DecodeAndVerify takes a license resource and will verify and decode the claims
// It returns the decoded LicenseClaims and a bool indicating if the license is valid.
func DecodeAndVerify(lic api.LicenseKey) (LicenseClaims, bool) {
	tok, err := jwt.ParseSignedAndEncrypted(lic.Spec.Token)
	if err != nil {
		log.Errorf("error parsing license: %s", err)
		return LicenseClaims{}, false
	}

	nested, err := tok.Decrypt(symKey)
	if err != nil {
		log.Errorf("error decrypting license: %s", err)
		return LicenseClaims{}, false
	}

	cert, err := cryptolicensing.LoadCertFromPEM([]byte(lic.Spec.Certificate))
	if err != nil {
		log.Errorf("error loading license certificate: %s", err)
		return LicenseClaims{}, false
	}

	var claims LicenseClaims
	if err := nested.Claims(cert.PublicKey, &claims); err != nil {
		log.Errorf("error parsing license claims: %s", err)
		return LicenseClaims{}, false
	}

	// Check if the license is expired.
	expired := claims.Claims.Expiry.Time().After(time.Now().UTC())

	return claims, expired
}