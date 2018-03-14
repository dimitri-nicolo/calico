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
	// CustomerID is a unique UUID assigned for each customer license.
	CustomerID          string   `json:"customer_id"`

	// Node count is not enforced in v2.1.
	Nodes       int      `json:"nodes" validate:"required"`

	// Name is name of the customer, so we can use the same name for multiple
	// licenses for a customer, but they'd have different CustomerID.
	Name        string   `json:"name" validate:"required"`

	// Features field is for future use.
	Features    []string `json:"features"`

	// GracePeriod is how many days the cluster will keep working even after
	// the license expires. This defaults to 90 days.
	// Currently not enforced.
	GracePeriod int      `json:"grace_period"`

	// Term is the actual license term in days.
	Term        int      `json:"term"`

	// Offline field is not used in v2.1.
	Offline     bool     `json:"offline"`

	// Include the default JWT claims.
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
	expired := claims.Claims.NotBefore.Time().After(time.Now().UTC())

	return claims, expired
}