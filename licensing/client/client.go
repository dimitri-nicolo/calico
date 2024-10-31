package client

import (
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/licensing/client/features"
	cryptolicensing "github.com/projectcalico/calico/licensing/crypto"
)

// Tigera entitlements root CA cert.
// All future licenseKeys' certificates will be signed by this
// cert's private key and can be verified using this certificate.
const rootPEM = `-----BEGIN CERTIFICATE-----
MIIGVDCCBDygAwIBAgIRALEvIBUYzHtatmXRoFZFvSkwDQYJKoZIhvcNAQELBQAw
gawxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRYwFAYDVQQHEw1T
YW4gRnJhbmNpc2NvMRQwEgYDVQQKEwtUaWdlcmEsIEluYzEiMCAGA1UECwwZU2Vj
dXJpdHkgPHNpcnRAdGlnZXJhLmlvPjE2MDQGA1UEAxMtVGlnZXJhIFNlbGYtU2ln
bmVkIFJvb3QgQ2VydGlmaWNhdGUgQXV0aG9yaXR5MB4XDTE4MDQwNTIxMjk0NVoX
DTI4MDQxMjIxMjk0MlowgbUxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9y
bmlhMRYwFAYDVQQHEw1TYW4gRnJhbmNpc2NvMRQwEgYDVQQKEwtUaWdlcmEsIElu
YzEiMCAGA1UECwwZU2VjdXJpdHkgPHNpcnRAdGlnZXJhLmlvPjE/MD0GA1UEAxM2
VGlnZXJhIEVudGl0bGVtZW50cyBJbnRlcm1lZGlhdGUgQ2VydGlmaWNhdGUgQXV0
aG9yaXR5MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAyxR6cSweVsw1
TDPU2Oo7MQIIZczguKAdjW7dCx+AIX+08NzmPpsV9xqiGGGoKnNsxzCz1QHPV+h3
AbWo7GlOrlGVITpad6igXJ5oE0sfZZprt1fvKY/ZE+1n31b6SV5FOYdn5prL5hUq
/xI55jQrayXTVjRq8B8JfC2pLOs0dkxLC6+HoGkyWE+uvfS+svGdGjLVO4P+TqIA
YNGPkSfn7fUI+ilbsS0F3YQ2prKcqvxkoMO6RlOJw4Zh8HXSJuq4sIqRY2KxWGLN
QQX46NJTzFLylPz0h4Tud3P49eHcgr2WQpepgclTxFxtOi3TTSoMwfg+bAgHsoJm
qDXiYgmq0D35gPAI4ElIBqCoGGaFhctwlpMpbuwYwmxSYE6xkVdQoxnueR4qgGCv
31WpoaEqOTVtiZuLppnVxofP0IiKVoB8GuliWz5BkH/aWpXbnGf13WoVIAZSVyfu
aE60/Jnm9AL5yEAJSErVrU82kHvaWwZ7vexRwj6PpRc/MT7zidUdfZqXijeUl+rx
o02ToOzYYri5HU3kCG5B/Gut8FIVt3oiNhwEFq0lBX6s6YYLdWo32lMNM4t5yong
j5O2qKGd4mZpuGQVjyvV210EvBGsRBgUy2tDcmCzvibZaQMclOayLbKmlogRp9B6
da9NNOgTY83w8OtVnudwM97vtkZLLU0CAwEAAaNmMGQwDgYDVR0PAQH/BAQDAgGG
MBIGA1UdEwEB/wQIMAYBAf8CAQAwHQYDVR0OBBYEFMWQOZIn86ODZ4kHxim/uMK7
kyBaMB8GA1UdIwQYMBaAFLBHvSxhIUK0KGQcRVLYs9mEfB/nMA0GCSqGSIb3DQEB
CwUAA4ICAQAXYQY3O2BtdHI50YGe7owxXTpYTg9tt6/2rAgdU9KpquyfLzH1qTO+
ZafW6/lXZdrMXgI19Txl9RBPw2DwQJIFYRpFFuQfkHiJumNtDFKt+PH13ZiukZid
+kei1ktoxxRzzpa8Rks2b70fruwfBmx7/KCKFEUcbdJGfpdy6yIgtVcPkBz/y6Qw
66Wopw61fF9Kr8Ugat18V/OIlOR1ErsoKA2GHohkTDwIHLMEXRArR/st3FnTMpb5
gIaNtObfh72ugVXhWslDC3CkIGtGLK/zak/LTqNwFeuYFTGE6op4jtocV7zQzJWx
POSjWs8qhHjhRle3DWwwooz4CfCFSWHWiaa6G4MXyY3+VNoG+0Gse3709ut45zjm
8cuyIHuPbbRO/Pr2Q1mplZdRqDxXIVAssiS/vwSieC+7jYeIDoXvFb6BRnSjdQuF
b8wF262rMiZlYhWlrbTYMTlyymxLp4H3nZ/V2XjFFWtC4iI4ImvPCUJRFtPZvwIR
r4MoQ19OK0Mqny70PbosS9FI3wIfkPm0Ih4VrecAlMnvpo6wuYV/kONx6DaUGr7s
ZVOtL6cv8afMyoaAQ/+O4xCd6bkCUnZe9DRp7AFfns6tY64MMvft/34Z1q0OyE01
6YA0CdAuqhS47spN26cyJUfj+JTO9rqnHvSy+h7QZgBQCu16a6ZEIA==
-----END CERTIFICATE-----`

var (
	// Symmetric key to encrypt and decrypt the JWT.
	// It has to be 32-byte long UTF-8 string.
	symKey = []byte("i༒2ஹ阳0?!pᄚ3-)0$߷५ૠm")

	// LicenseKey is a singleton resource, and has to have the name "default".
	ResourceName = "default"

	//opts x509.VerifyOptions
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

	//opts = x509.VerifyOptions{
	//	Roots: roots,
	//}
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
	// We will default this with `[ “cnx”, “all”]` for v2.1 and Enterprise package
	// Cloud licenses will have one of the following values: ["cloud", "community", ...],
	// ["cloud", "starter", ...] or [ "cloud", "pro", ...]. Individual features are appended after the license
	// package.
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
	tok, err := jwt.ParseSignedAndEncrypted(
		lic.Spec.Token,
		[]jose.KeyAlgorithm{jose.A128GCMKW},
		[]jose.ContentEncryption{jose.A128GCM},
		[]jose.SignatureAlgorithm{jose.PS512})
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

	rootCert, err := cryptolicensing.LoadCertFromPEM([]byte(rootPEM))
	if err != nil {
		return LicenseClaims{}, fmt.Errorf("error loading license certificate: %s", err)
	}

	// We only check if the certificate was signed by Tigera root certificate.
	// Verify() also checks if the certificate is expired before checking if it was signed by the root cert,
	// which is not what we want to do for v2.1 behavior.
	if err = cert.CheckSignatureFrom(rootCert); err != nil {
		return LicenseClaims{}, fmt.Errorf("failed to verify the certificate: %s", err)
	}

	// For v2.1 we are not checking certificate expiration, verifying the cert chain also checks if the leaf certificate
	// is expired, and since we don't really stop any features from working after the license expires (at least in v2.1)
	// We have to deal with a case where the certificate is expired but license is still "valid" - i.e. within the grace period
	// which could be max int.
	// We can uncomment this when we actually enforce license and stop the features from running.
	//if _, err := cert.Verify(opts); err != nil {
	//	return LicenseClaims{}, fmt.Errorf("failed to verify the certificate: %s", err)
	//}

	var claims LicenseClaims
	if err := nested.Claims(cert.PublicKey, &claims); err != nil {
		return LicenseClaims{}, fmt.Errorf("error parsing license claims: %s", err)
	}

	return claims, nil
}

// IsOpenSourceAPI determines is a calico API is defined as an open
// source API
func IsOpenSourceAPI(resourceGroupVersionKind string) bool {
	return features.OpenSourceAPIs[resourceGroupVersionKind]
}

// IsManagementAPI determines is a calico API is defined as an api used to managed/access
// resources on a calico install
func IsManagementAPI(resourceGroupVersionKind string) bool {
	return features.ManagementAPIs[resourceGroupVersionKind]
}

// ErrExpiredButWithinGracePeriod indicates the license has expired but is within the grace period.
type ErrExpiredButWithinGracePeriod struct {
	Err error
}

func (e ErrExpiredButWithinGracePeriod) Error() string {
	return "license expired"
}

type LicenseStatus int

const (
	Unknown LicenseStatus = iota
	Valid
	InGracePeriod
	Expired
	NoLicenseLoaded
)

func (s LicenseStatus) String() string {
	switch s {
	case Valid:
		return "valid"
	case InGracePeriod:
		return "in-grace-period"
	case Expired:
		return "expired"
	case NoLicenseLoaded:
		return "no-license-loaded"
	default:
		return "unknown"
	}
}

// Validate checks if the license is expired.
func (c *LicenseClaims) Validate() LicenseStatus {
	return c.ValidateAtTime(time.Now())
}

// Validate checks if the license is expired.
func (c *LicenseClaims) ValidateAtTime(t time.Time) LicenseStatus {
	if c == nil {
		return NoLicenseLoaded
	}

	expiryTime := c.Claims.Expiry.Time()
	if expiryTime.After(t) {
		return Valid
	}

	gracePeriodExpiryTime := expiryTime.Add(time.Duration(c.GracePeriod) * time.Hour * 24)
	if gracePeriodExpiryTime.After(t) {
		return InGracePeriod
	}

	return Expired
}

// ValidateFeature returns true if the feature is enabled, false if it is not.
// False is returned if the license is invalid in any of the following ways:
// - there isn't a license
// - the license has expired and is no longer in its grace period.
func (c *LicenseClaims) ValidateFeature(feature string) bool {
	return c.ValidateFeatureAtTime(time.Now(), feature)
}

// ValidateFeature returns true if the feature is enabled, false if it is not.
// False is returned if the license is invalid in any of the following ways:
// - there isn't a license
// - the license has expired and is no longer in its grace period.
func (c *LicenseClaims) ValidateFeatureAtTime(t time.Time, feature string) bool {
	switch c.ValidateAtTime(t) {
	case NoLicenseLoaded, Expired:
		return false
	}

	if len(c.Features) == 0 {
		return false
	}

	var licensePackage = strings.Join(c.Features, "|")

	switch licensePackage {
	case features.Enterprise:
		// This is maintain backwards compatibility for any cloud license issued for 3.5
		return true
	case features.CloudCommunity:
		// This is maintain backwards compatibility for any cloud license issued for 3.5
		return features.CloudCommunityFeatures[feature]
	case features.CloudStarter:
		// This is maintain backwards compatibility for any cloud license issued for 3.5
		return features.CloudStarterFeatures[feature]
	case features.CloudPro:
		// This is maintain backwards compatibility for any cloud license issued for 3.5
		return features.CloudProFeatures[feature]
	default:
		for _, f := range c.Features {
			if f == features.All {
				return true
			}
			if f == feature {
				return true
			}
		}

	}

	return false
}

// ValidateAPIUsage checks if the API can be accessed.
func (c *LicenseClaims) ValidateAPIUsage(gvk string) bool {
	if IsOpenSourceAPI(gvk) || IsManagementAPI(gvk) {
		return true
	}

	feature, ok := features.EnterpriseAPIsToFeatureName[gvk]
	if ok {
		return c.ValidateFeature(feature)
	}

	return false
}
