package types

//import "gopkg.in/square/go-jose.v2/jwt"
//
//// LicenseClaims contains all the license control fields.
//// This includes custom JWT fields and the default ones.
//type LicenseClaims struct {
//	// CustomerID is a unique UUID assigned for each customer license.
//	CustomerID          string   `json:"customer_id"`
//
//	// Node count is not enforced in v2.1.
//	Nodes       int      `json:"nodes" validate:"required"`
//
//	// Name is name of the customer, so we can use the same name for multiple
//	// licenses for a customer, but they'd have different CustomerID.
//	Name        string   `json:"name" validate:"required"`
//
//	// Features field is for future use.
//	Features    []string `json:"features"`
//
//	// GracePeriod is how many days the cluster will keep working even after
//	// the license expires. This defaults to 90 days.
//	// Currently not enforced.
//	GracePeriod int      `json:"grace_period"`
//
//	// Term is the actual license term in days.
//	Term        int      `json:"term"`
//
//	// Offline field is not used in v2.1.
//	Offline     bool     `json:"offline"`
//
//	// Include the default JWT claims.
//	jwt.Claims
//}
//
//
//// License contains signed and encrypted JWT (aka JWS & JWE - nested)
//// as well as the certificate signed by Tigera root certificate to
//// verify if the license was issued by Tigera.
//type License struct {
//	Claims string `json:"claims" yaml:"claims"`
//	Cert   string `json:"cert" yaml:"cert"`
//}