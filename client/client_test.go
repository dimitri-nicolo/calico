package client_test

import (
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/satori/go.uuid"
	"gopkg.in/square/go-jose.v2/jwt"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/davecgh/go-spew/spew"

	api "github.com/projectcalico/libcalico-go/lib/apis/v3"
	"github.com/tigera/licensing/client"
)

var (
	cid1 string

	testPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAujFhL4jgF4Xns21Jeg0yyqxwXPdHwHZ28zKphd3hrJQqS4qs
JJtPOp6Lxf+BRHWI/E9KC4Zz5mxLWswApD8ge3MkgE9v6ovfPbJ+G/X6jJtyVKt1
9LpoapYJPAHDqKU+kzSp5bL1i7Vo6Ipye9GlJ0ZjdTEMbi9BNna4z9L+fktDl5mg
iBqR4L5eMtY5cFS1cqepeMcZY0WVSD2JpoNv8Sqoeqh735v/fDFhN1yyPIEhL+TN
vpp58So47OMcymzqaM4YJPbPunMIo9Rn4oGvfh5I42cxRAzOOKKeRiGnLObqCyZL
wP3P3h8uXKKrYtsBt4IKHV5tJglN+wilonIzlwIDAQABAoIBAFJr13ymV6SyFv47
a6JGw2wqZ1cP88hD6KYBkD99GBBASnTEPy25PppRYshUMZHvgaNHKhzt+NJQsA7S
bZpHg4aCUu8luwIVxs3V/LM98RpbGYJXoFCkT+KW5iGVGlrGQ2wAjRDsZnvg4z7F
QqaDCFvZcd+HxdvkuTZ12ZvN5/BZeOD/Thdvm8B7xTRIqv0qVkrGdV5XyMaa7tQq
g5cOYRD04iqcBQ14uIbm0/o008ovmuThpEG+Pc1hC5EqY5QcXYwlRfCKdJ9bwCtX
2Ar9l3ThVAjeAQkviv6Cq45j52sSc/rh4rayc9WaJl8vznc+SQ6Wq8JQg3AFURXz
IjJeXRkCgYEA7fTt8eN/vrYSOvj8BM+lkRozFMYmxHAJWwNNSGzCmVwPw4zzy4ND
Zw7ubMzKAMhi3h48QhZXQSe7a/O8jBtOM5Twx8ngahgTCXwlmXtO1PPztUt773Td
07JJuLIp42DssXGuLijnCnYteCUIhFu6PyHgJsbkEcPaLctWIVDRjhUCgYEAyE+k
SWVdNl4zjF3G55ErEwpK4FB9lvp9ybKV4QuRRkSTU429ljOLd5hbIl0iq2Qm1GHb
bwcuNx9Wj5uNZXMf3UUyH3ZcFuNuzJLaZQRF+Ms6tT9SweZ5cd0DWF866P//WPa+
Gq1NC1iyL7KQ/LTTlKWJOuRcSUD7C0wHr+f6kfsCgYEAv0C25k2VZPDtohxwYmWK
ix9lovLIQeZSfqYevXE8zwohWWi2ogG0cOadVzEZwptMa34drHhMVP/cMZ3LE0j3
B5pUFB/7kQoccuknRz7GU35niHVM/V8O05Fek2YPKMPEObJG7q7NU6k8Tm5ldAxN
m2Rcxo3gzS5+84OUjF5qrykCgYEAwzHQnvEW1x8Ozm6noAo3VlOGSXZGG/S21PCg
2u8Bvt6eTiJmJ9LMylr+G8t0OF3c9MLzKQtvPqncGQ70x3JbD60ZPc2ByZAQ7WsB
RMTYRqwL5ojxZR/pIkrDsr8B0gF8W7393FMaK79fy9kPLiIrt8Njqa7UO1IGEKkj
KIg/BTcCgYB9kUKCO8wA3tv9yQvOmIJfP7gQ1R4+C0SQqXz7UJXMY3x8jgCOnI/6
lCzU+OFviVtLdml43hyiGfF1z2RkQbZ4n3nChiG8ofN1wxtCQSSJUJBitc+0Hj24
I/RHizr6nquNwLgXyVzezzzLU08zc0Bjy9RPuqxB0A2qduCBAYvxnw==
-----END RSA PRIVATE KEY-----`

	testCert = `-----BEGIN CERTIFICATE-----
MIIDSDCCAjCgAwIBAgIITWWCIQf8/VIwDQYJKoZIhvcNAQELBQAwKjEUMBIGA1UE
ChMLVGlnZXJhIEluYy4xEjAQBgNVBAMTCXRpZ2VyYS5pbzAeFw0xODAzMTMwMDM3
MDRaFw0xOTAzMTMwMDM3MDRaMCoxFDASBgNVBAoTC1RpZ2VyYSBJbmMuMRIwEAYD
VQQDEwl0aWdlcmEuaW8wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQC6
MWEviOAXheezbUl6DTLKrHBc90fAdnbzMqmF3eGslCpLiqwkm086novF/4FEdYj8
T0oLhnPmbEtazACkPyB7cySAT2/qi989sn4b9fqMm3JUq3X0umhqlgk8AcOopT6T
NKnlsvWLtWjoinJ70aUnRmN1MQxuL0E2drjP0v5+S0OXmaCIGpHgvl4y1jlwVLVy
p6l4xxljRZVIPYmmg2/xKqh6qHvfm/98MWE3XLI8gSEv5M2+mnnxKjjs4xzKbOpo
zhgk9s+6cwij1Gfiga9+HkjjZzFEDM44op5GIacs5uoLJkvA/c/eHy5coqti2wG3
ggodXm0mCU37CKWicjOXAgMBAAGjcjBwMA4GA1UdDwEB/wQEAwICpDAPBgNVHRMB
Af8EBTADAQH/MA0GA1UdDgQGBAQBAgMEMD4GA1UdEQQ3MDWCCXRpZ2VyYS5pb4IT
bGljZW5zaW5nLnRpZ2VyYS5pb4ETbGljZW5zaW5nQHRpZ2VyYS5pbzANBgkqhkiG
9w0BAQsFAAOCAQEAqJx4TL848wlaUaYmkz7nOEJDp5rsZRcHauPAAfBzl7+P6v5/
4BVkQDnE6t0MCpK5T3JgyOaWQeK6fpmMmkfIL4H7Za/OPivU9QM0ir0XJyMebiP4
UlunYrcwVU+PiGEmQxNeMGlhaxXQvqs6vWlzRFTYAoDYjV78WzskqLxypVzKLvyd
GVf0wBlyvsDXDr1fpqizH2VECzLgmzqgAKnAWrj4Hm/JOlb0SBoXGbpWKq+/FxPc
lB/9gAlWPAhPQi5DTliwWoAm9JWIcszk+JG3a8N3Fnp6P4u5TgjcNktk9rU9hOPW
CrsfFSIo6is9W3G+E+7LcsZySLji8JatxslsGg==
-----END CERTIFICATE-----
`

	evilCert = `-----BEGIN CERTIFICATE-----
MIIDSDCCAjCgAwIBAgIITWWCIQf8/VIwDQYJKoZIhvcNAQELBQAwKjEUMBIGA1UE
ChMLVGlnZXJhIEluYy4xEjAQBgNVBAMTCXRpZ2VyYS5pbzAeFw0xODAzMTkxODQ1
MzZaFw0yMzAzMTgxODQ1MzZaMCoxFDASBgNVBAoTC1RpZ2VyYSBJbmMuMRIwEAYD
VQQDEwl0aWdlcmEuaW8wggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCu
u2TJTR5L0fav6P0X7KRT2/ttxMJZ3+nSn4moebeM88SzsDdFjkFbE1mVYy2Sf5lN
zPxYhfTwxHPPawXhaSRsDqYKDpOgpP3phEUsubsRTXOPtrYLUbfjDMcgpnHStQSm
gX1jccVBfM2FxjY9Tb4DRdNHax0cwgKDfeo4+FhwQDiK+zykttdDM01T6ca/3OsU
yxihbf8HPkKRiWLpvZ1QyifyrIPOstw+gmRyGGvOIIDXOM6s+RVQVJqSkbTOXAbx
4km3+CSckoewOOe/O0kL7dM7zLxKN00SOpJ4cWa+D4eG4bYNiLtR+nKHBcAmdeYQ
fgZo8V1N/RR1H/8ueQsnAgMBAAGjcjBwMA4GA1UdDwEB/wQEAwICpDAPBgNVHRMB
Af8EBTADAQH/MA0GA1UdDgQGBAQBAgMEMD4GA1UdEQQ3MDWCCXRpZ2VyYS5pb4IT
bGljZW5zaW5nLnRpZ2VyYS5pb4ETbGljZW5zaW5nQHRpZ2VyYS5pbzANBgkqhkiG
9w0BAQsFAAOCAQEAeKuGfizTKtSa6D/0XO+9MHKC8GGsSdf9KKy9ZTHG2W/fiYIn
o0FUoymcr+Dnv9KziH5TF2+YtvAKYIzoREoWIfqL1RmDsyBcLNRsOf8OtXjfFyrF
rUc7uMYa/IhE505K6Q/4636BXb/+/hr0PnACW2x+rO26bLu1h0Ky95eqcwKoewNr
batmSSXOYT9ysh1kfcDJW2ky4aruz7va2/oCsHjTltaBkh3O3VIpqBeHa+i4qxB1
SSsNpgNEXVNjF07I4OdHVdyWyWezqAPWFRdairdhXrHVqO3EJ05Ik83sO0ZCNLwe
XJiNPmt5C55ETI7JUGZM466nO/ymmfMo0feFZw==
-----END CERTIFICATE-----
`

	// Tigera private key location.
	pkeyPath = "../test-data/test_tigera.io_private_key.pem"

	// Tigera license signing certificate path.
	certPath = "../test-data/test_tigera.io_certificate.pem"

	absPkeyPath, absCertPath string
)

func init() {
	absPkeyPath, _ = filepath.Abs(pkeyPath)
	absCertPath, _ = filepath.Abs(certPath)
}

var claimToJWTTable = []struct {
	description string
	claim       client.LicenseClaims
}{
	{
		description: "fully populated claim",
		claim: client.LicenseClaims{
			CustomerID:  uuid.NewV4().String(),
			Nodes:       420,
			Name:        "meepster-inc",
			Features:    []string{"nice", "features", "for", "you"},
			GracePeriod: 88,
			Offline:     true,
			Claims: jwt.Claims{
				NotBefore: jwt.NewNumericDate(time.Date(2022, 3, 14, 23, 59, 59, 999999999, time.Local)),
				IssuedAt:  jwt.NewNumericDate(time.Now().Local()),
			},
		},
	},
	{
		description: "only required fields for v2.1 populated",
		claim: client.LicenseClaims{
			CustomerID:  uuid.NewV4().String(),
			Nodes:       555,
			Name:        "cool-cat-inc",
			GracePeriod: 90,
			Offline:     true,
			Claims: jwt.Claims{
				NotBefore: jwt.NewNumericDate(time.Date(2022, 3, 14, 23, 59, 59, 999999999, time.Local)),
			},
		},
	},
	{
		description: "partially populated claim",
		claim: client.LicenseClaims{
			CustomerID: uuid.NewV4().String(),
			Nodes:      1000,
			Name:       "lame-banana-inc",
			Claims: jwt.Claims{
				NotBefore: jwt.NewNumericDate(time.Date(2021, 3, 14, 23, 59, 59, 999999999, time.Local)),
			},
		},
	},
}

func TestGetLicenseFromClaims(t *testing.T) {
	for _, entry := range claimToJWTTable {
		t.Run(entry.description, func(t *testing.T) {
			RegisterTestingT(t)

			lic, err := client.GetLicenseFromClaims(entry.claim, absPkeyPath, absCertPath)

			spew.Dump(time.Now().Local())

			// We cannot assert the token because it's hard to generate the exact random feed used to encrypt the JWT.
			Expect(err).NotTo(HaveOccurred(), entry.description)

			// We can verify the generated resource's Objectmeta name.
			Expect(lic.Name).Should(Equal("default"), entry.description)

			claims, valid := client.DecodeAndVerify(*lic)
			Expect(valid).Should(BeTrue(), entry.description)
			Expect(claims).Should(Equal(entry.claim), entry.description)
		})
	}
}

var tokenToLicense = []struct {
	description string
	license     api.LicenseKey
	claim       client.LicenseClaims
	valid       bool
}{
	{
		description: "fully populated claim",
		license: api.LicenseKey{
			ObjectMeta: v1.ObjectMeta{
				Name: "default",
			},
			Spec: api.LicenseKeySpec{
				Token:       "eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJ3WWpuYjV6TTF5MlV6RFZ4IiwidGFnIjoiN1dSUkxPanNGQ0F1R3pNRGg5akc1USIsInR5cCI6IkpXVCJ9.HtXrz5-Q_vVfKwgn9Ig_zQ.xf6FZYH3315Tffzv.v7JNl7qOWTivF3Y0Fla-5uG-SM7zCVWcOWEncS7y5kc_uIIRTvTqXV7LAB0b6rZFkXGYxo3X0nBADh7yVJO2S9LX3AbjhF4g_5Vu1uVHwNyKEmSxoMhJGK8v0kwtmXWF7dgICKlAWcSE2kscr-1P-m-MgjTPIZaQU27EN3KFNBgPtLalSKcTRoKMWbqnZRyZFB4gIhpXRKOi2wSlRwbzflumRt5PBGQ6AAdqJaZhEDKYIRVwiYiLh8ODXC2WNhF9KS7GqXRE9QopOcQkh3n_AAADIgzOMdrVr26VTXKXZlwtTYZ5cNPxRZA7QkQVB9HMh7WwwstcSLlVRnHcGZJwmTUfpdGExAywCu4DkqJRnarfJUmG1Y86ecOFnmuycFo0NPuruUEXUG33Nd_670qOWzICjqu68cx3AXcwh46m8hZGR3Zbs1usYfrWTVfFZxNUYlAOCmjrnIAKfxDe4B4fBKYEyFM7PTUQj1UTChgv5G3wRBZiVPDv67gnOrqtQQNyAtJvWsaSdxEu5LGzO68ntauYM4wohnqx4JBzFrd5YkWivHf10yFb7_mGYxhqG7_lPiWAd7zxJNGYrOHi8qEMPFtKANI4UKLAbyXVgPJuTo_kAmoHpSqvAf2DTNODBJQb_hl6F6gX0gWsJIQ1V7O7xn6aAc0nkiizYSLuoKLSsF8rWSyASnPuHhc5AeFVEqA8oRYeZLMh9BBYr8w3kGa6eobtp8j8g2YcEy-KSCgxuef94OIRn6EPbvkfhhz8bZm9c1670N701J91WnIG7l1WXFAxXnfO055W0ulpbE99sw.HACGOFtKA6ZvoAg4Prgiaw",
				Certificate: testCert,
			},
		},

		claim: client.LicenseClaims{
			CustomerID:  "meow23424coldcovfefe0nmyfac3",
			Nodes:       420,
			Name:        "meepster-inc",
			Features:    []string{"nice", "features", "for", "you"},
			GracePeriod: 88,
			Offline:     true,
			Claims: jwt.Claims{
				NotBefore: jwt.NewNumericDate(time.Date(2020, 3, 14, 23, 59, 59, 59, time.Local)),
			},
		},

		valid: true,
	},
	{
		description: "claim with the JWT header meddled with",
		license: api.LicenseKey{
			ObjectMeta: v1.ObjectMeta{
				Name: "default",
			},
			Spec: api.LicenseKeySpec{
				Token:       "eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJ3WWpuYjV6TTF5MlV6RFZ4IiwidFnIjoiN1dSUkxPanNGQ0F1R3pNRGg5akc1USIsInR5cCI6IkpXVCJ9.HtXrz5-Q_vVfKwgn9Ig_zQ.xf6FZYH3315Tffzv.v7JNl7qOWTivF3Y0Fla-5uG-SM7zCVWcOWEncS7y5kc_uIIRTvTqXV7LAB0b6rZFkXGYxo3X0nBADh7yVJO2S9LX3AbjhF4g_5Vu1uVHwNyKEmSxoMhJGK8v0kwtmXWF7dgICKlAWcSE2kscr-1P-m-MgjTPIZaQU27EN3KFNBgPtLalSKcTRoKMWbqnZRyZFB4gIhpXRKOi2wSlRwbzflumRt5PBGQ6AAdqJaZhEDKYIRVwiYiLh8ODXC2WNhF9KS7GqXRE9QopOcQkh3n_AAADIgzOMdrVr26VTXKXZlwtTYZ5cNPxRZA7QkQVB9HMh7WwwstcSLlVRnHcGZJwmTUfpdGExAywCu4DkqJRnarfJUmG1Y86ecOFnmuycFo0NPuruUEXUG33Nd_670qOWzICjqu68cx3AXcwh46m8hZGR3Zbs1usYfrWTVfFZxNUYlAOCmjrnIAKfxDe4B4fBKYEyFM7PTUQj1UTChgv5G3wRBZiVPDv67gnOrqtQQNyAtJvWsaSdxEu5LGzO68ntauYM4wohnqx4JBzFrd5YkWivHf10yFb7_mGYxhqG7_lPiWAd7zxJNGYrOHi8qEMPFtKANI4UKLAbyXVgPJuTo_kAmoHpSqvAf2DTNODBJQb_hl6F6gX0gWsJIQ1V7O7xn6aAc0nkiizYSLuoKLSsF8rWSyASnPuHhc5AeFVEqA8oRYeZLMh9BBYr8w3kGa6eobtp8j8g2YcEy-KSCgxuef94OIRn6EPbvkfhhz8bZm9c1670N701J91WnIG7l1WXFAxXnfO055W0ulpbE99sw.HACGOFtKA6ZvoAg4Prgiaw",
				Certificate: testCert,
			},
		},

		claim: client.LicenseClaims{},

		valid: false,
	},
	{
		description: "claim with the JWT payload meddled with",
		license: api.LicenseKey{
			ObjectMeta: v1.ObjectMeta{
				Name: "default",
			},
			Spec: api.LicenseKeySpec{
				Token:       "eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJ3WWpuYjV6TTF5MlV6RFZ4IiwidGFnIjoiN1dSUkxPanNGQ0F1R3pNRGg5akc1USIsInR5cCI6IkpXVCJ9.HtXrz5-Q_vVfKwgn9Ig_zQ.xf6FZYH3315Tffzv.v7JNl7qOWTivFY0Fla-5uG-SM7zCVWcOWEncS7y5kc_uIIRTvTqXV7LAB0b6rZFkXGYxo3X0nBADh7yVJO2S9LX3AbjhF4g_5Vu1uVHwNyKEmSxoMhJGK8v0kwtmXWF7dgICKlAWcSE2kscr-1P-m-MgjTPIZaQU27EN3KFNBgPtLalSKcTRoKMWbqnZRyZFB4gIhpXRKOi2wSlRwbzflumRt5PBGQ6AAdqJaZhEDKYIRVwiYiLh8ODXC2WNhF9KS7GqXRE9QopOcQkh3n_AAADIgzOMdrVr26VTXKXZlwtTYZ5cNPxRZA7QkQVB9HMh7WwwstcSLlVRnHcGZJwmTUfpdGExAywCu4DkqJRnarfJUmG1Y86ecOFnmuycFo0NPuruUEXUG33Nd_670qOWzICjqu68cx3AXcwh46m8hZGR3Zbs1usYfrWTVfFZxNUYlAOCmjrnIAKfxDe4B4fBKYEyFM7PTUQj1UTChgv5G3wRBZiVPDv67gnOrqtQQNyAtJvWsaSdxEu5LGzO68ntauYM4wohnqx4JBzFrd5YkWivHf10yFb7_mGYxhqG7_lPiWAd7zxJNGYrOHi8qEMPFtKANI4UKLAbyXVgPJuTo_kAmoHpSqvAf2DTNODBJQb_hl6F6gX0gWsJIQ1V7O7xn6aAc0nkiizYSLuoKLSsF8rWSyASnPuHhc5AeFVEqA8oRYeZLMh9BBYr8w3kGa6eobtp8j8g2YcEy-KSCgxuef94OIRn6EPbvkfhhz8bZm9c1670N701J91WnIG7l1WXFAxXnfO055W0ulpbE99sw.HACGOFtKA6ZvoAg4Prgiaw",
				Certificate: testCert,
			},
		},

		claim: client.LicenseClaims{},

		valid: false,
	},
	{
		description: "claim with the JWT signed by some evil random private key",
		license: api.LicenseKey{
			ObjectMeta: v1.ObjectMeta{
				Name: "default",
			},
			Spec: api.LicenseKeySpec{
				Token:       "eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJVeG1hUnBucS1Oc2JORWY1IiwidGFnIjoiZkFnR2I0U2R5WlRTTWJVTFZSVE91dyIsInR5cCI6IkpXVCJ9.ao8K2OAgme4kwVejNn-Lvg.CyEws8QbrDGjtkVx.Pebt9PmCpWvPcVYkzkY2BSP92RGCOfg7oGHSfo5MiabnXXn6KDQ6rT2wxHjcHTNcszYO8nZ_w4nUIvH0Vg-7VAbHhvYFpsbtuc8eXRSbqV9Vt0-jm4N9iQFbT5bEi-qyPk5p-OjK_UO8tAPll7foQz9DlqG1h55Pn2RyrjL2-oTJeDb5b7uRkLFASeD-ApqB6NylQ6oskCr9GN5vHaV5_tRaoaWTlCPFwUIQc1TMwoBDoyNTWJUV45QeuT6ha1T4IgiDS7uJcvPb7omm7dhoXK5aw-b-G8wVlWbfD-0ygzPr9qehkh9IYmJAQtYo46dTJBKIInQUss-IpURNUQKVuYrODFkw4GEpQ4FQAamIktYt_EHudzMrrtJM3xhvtYT9bYJz-0_wYnloy7kJMd7JHPaRxH3wICAw0UUe-0F8sViA5NTnADKSXnpWRRDArsFKezywdUqCgRV9lwHbaDKSJFaMSOMJ3BmTXOz_vJ1hiWCjelAUU0sE6r0tcIYPgc705hLYnRb5Xk_qePhtFdAZkqRkymnYJVRRYmQhVYaDEB33E9UYFLqL1EOhkfRnu-iNuMky9OfjuwrjoBaVJDlBQ9y76iOMoDZr4hpEIsESli8nY0MzzHLc2T4WUd1rx9XSw7VaojSYPvpK9JWhJkWcQVb28FNJB6Fui7V_T1bnF44vBqy2OKY3iK-OotULdm76Jm_rSXgpoJldUOjc31f6qTD78SeZ5UhyxgLGCzS5lHri1FCiYDjy6dcFGNfoWJ1Lpj5mTY_4OLnfLG2yqlyqRfrX8bTq5X0.V1LdXb0VgrJDlkeQ95GWmQ",
				Certificate: testCert,
			},
		},

		claim: client.LicenseClaims{},

		valid: false,
	},
	{
		description: "claim with the JWT signed by tigera but certificate is swapped out with an evil certificate",
		license: api.LicenseKey{
			ObjectMeta: v1.ObjectMeta{
				Name: "default",
			},
			Spec: api.LicenseKeySpec{
				Token:       "eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJ3WWpuYjV6TTF5MlV6RFZ4IiwidGFnIjoiN1dSUkxPanNGQ0F1R3pNRGg5akc1USIsInR5cCI6IkpXVCJ9.HtXrz5-Q_vVfKwgn9Ig_zQ.xf6FZYH3315Tffzv.v7JNl7qOWTivF3Y0Fla-5uG-SM7zCVWcOWEncS7y5kc_uIIRTvTqXV7LAB0b6rZFkXGYxo3X0nBADh7yVJO2S9LX3AbjhF4g_5Vu1uVHwNyKEmSxoMhJGK8v0kwtmXWF7dgICKlAWcSE2kscr-1P-m-MgjTPIZaQU27EN3KFNBgPtLalSKcTRoKMWbqnZRyZFB4gIhpXRKOi2wSlRwbzflumRt5PBGQ6AAdqJaZhEDKYIRVwiYiLh8ODXC2WNhF9KS7GqXRE9QopOcQkh3n_AAADIgzOMdrVr26VTXKXZlwtTYZ5cNPxRZA7QkQVB9HMh7WwwstcSLlVRnHcGZJwmTUfpdGExAywCu4DkqJRnarfJUmG1Y86ecOFnmuycFo0NPuruUEXUG33Nd_670qOWzICjqu68cx3AXcwh46m8hZGR3Zbs1usYfrWTVfFZxNUYlAOCmjrnIAKfxDe4B4fBKYEyFM7PTUQj1UTChgv5G3wRBZiVPDv67gnOrqtQQNyAtJvWsaSdxEu5LGzO68ntauYM4wohnqx4JBzFrd5YkWivHf10yFb7_mGYxhqG7_lPiWAd7zxJNGYrOHi8qEMPFtKANI4UKLAbyXVgPJuTo_kAmoHpSqvAf2DTNODBJQb_hl6F6gX0gWsJIQ1V7O7xn6aAc0nkiizYSLuoKLSsF8rWSyASnPuHhc5AeFVEqA8oRYeZLMh9BBYr8w3kGa6eobtp8j8g2YcEy-KSCgxuef94OIRn6EPbvkfhhz8bZm9c1670N701J91WnIG7l1WXFAxXnfO055W0ulpbE99sw.HACGOFtKA6ZvoAg4Prgiaw",
				Certificate: evilCert,
			},
		},

		claim: client.LicenseClaims{},

		valid: false,
	},
	{
		description: "claim with the JWT signed by an evil private key but certificate is still the tigera original cert",
		license: api.LicenseKey{
			ObjectMeta: v1.ObjectMeta{
				Name: "default",
			},
			Spec: api.LicenseKeySpec{
				Token:       "eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJLeWE0VHpEaWY2eFM3TTl2IiwidGFnIjoiYVhHR3d0alczSjhKeWgtb2hWajRJdyIsInR5cCI6IkpXVCJ9.LgbBH-IGmLH2iUFY171xwA.NImD2DVyH1ahbruT.DHhdADLX7BfwwYoknoTnPEQGh7vItF7YhYukfPDm_VlwgERXTDdqb6wFQQOZOvFFlcMRYBBzDQBguSkYEHYWegHIuZ7Amfh8uCcI0l93BPz1TrOZdX4fukikb5YVTbRJjxgJTvakucG9dh45hwks9gUCGdXFvVAJH_wMDc_kPVeb0fx84f_H30gNswvKItyIT09lOiRCfy9HOGdpo1RlA0UCZvIPYD9zSl1_ldGZ5Oj2RYz9HU7bhuqV4AU7OuglE_8yvNMmkqSD9BmiLOxzxMVvg3uj5trmuTOy4pAZuchykM3p-DgGiWuo4kyaHvpcfIISSyBU8xtVMyWALayeaschyvlAvRJHAVjKd9Cubx5akA23w4KpBGsJ2EgQPNmyHdEoxqKohO6KbYcOvsD7PThH8e9UV7GgGrQp4OUBZXfym-_yi_erI6FC91n3rgcSMqYpIrhC5-dPSExKuPVA_94dlcP-cDxAtuL8W0T8mafTqKl4Vg-Ojaj7pul4-i7223loZSbkYEpuoTzHYglgB2_PfHgkZsqgl8adlm7muKpxSe_TH-6wQh6fXxGzUJEu7DLvcy82r5v_HcWtJUj43qu8BTHR4sc4_1NU8eHya_HtwgvOo98Ze1Gd9qC_GOFkMYomEk2ogarPnGGKD-gfMN3GxziUz5d4kpb8mzknGIX5hqaxcslV4HDnSA97zjssyajg1Eh-a6xOIaPOlYW3YzXQ3GQPABLn18V2hFCNhB-ml6KWceYA6EsxnKqdEK2KN8dnDGESdjwCIUfcY7KFRD30qhAOUAKpU14.YvpAmE0JPK1Brn7kgGphlg",
				Certificate: testCert,
			},
		},

		claim: client.LicenseClaims{},

		valid: false,
	},
	{
		// TODO (gunjan5): THIS TEST SHOULD FAIL ONCE WE ADD CERT CHAIN VALIDATION!!!!
		description: "claim with the JWT with an evil cert and signed by an evil private key",
		license: api.LicenseKey{
			ObjectMeta: v1.ObjectMeta{
				Name: "default",
			},
			Spec: api.LicenseKeySpec{
				Token:       "eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJLeWE0VHpEaWY2eFM3TTl2IiwidGFnIjoiYVhHR3d0alczSjhKeWgtb2hWajRJdyIsInR5cCI6IkpXVCJ9.LgbBH-IGmLH2iUFY171xwA.NImD2DVyH1ahbruT.DHhdADLX7BfwwYoknoTnPEQGh7vItF7YhYukfPDm_VlwgERXTDdqb6wFQQOZOvFFlcMRYBBzDQBguSkYEHYWegHIuZ7Amfh8uCcI0l93BPz1TrOZdX4fukikb5YVTbRJjxgJTvakucG9dh45hwks9gUCGdXFvVAJH_wMDc_kPVeb0fx84f_H30gNswvKItyIT09lOiRCfy9HOGdpo1RlA0UCZvIPYD9zSl1_ldGZ5Oj2RYz9HU7bhuqV4AU7OuglE_8yvNMmkqSD9BmiLOxzxMVvg3uj5trmuTOy4pAZuchykM3p-DgGiWuo4kyaHvpcfIISSyBU8xtVMyWALayeaschyvlAvRJHAVjKd9Cubx5akA23w4KpBGsJ2EgQPNmyHdEoxqKohO6KbYcOvsD7PThH8e9UV7GgGrQp4OUBZXfym-_yi_erI6FC91n3rgcSMqYpIrhC5-dPSExKuPVA_94dlcP-cDxAtuL8W0T8mafTqKl4Vg-Ojaj7pul4-i7223loZSbkYEpuoTzHYglgB2_PfHgkZsqgl8adlm7muKpxSe_TH-6wQh6fXxGzUJEu7DLvcy82r5v_HcWtJUj43qu8BTHR4sc4_1NU8eHya_HtwgvOo98Ze1Gd9qC_GOFkMYomEk2ogarPnGGKD-gfMN3GxziUz5d4kpb8mzknGIX5hqaxcslV4HDnSA97zjssyajg1Eh-a6xOIaPOlYW3YzXQ3GQPABLn18V2hFCNhB-ml6KWceYA6EsxnKqdEK2KN8dnDGESdjwCIUfcY7KFRD30qhAOUAKpU14.YvpAmE0JPK1Brn7kgGphlg",
				Certificate: evilCert,
			},
		},

		claim: client.LicenseClaims{
			CustomerID:  "fda67a1c-1791-4157-8ddc-f11f265db0d0",
			Nodes:       555,
			Name:        "iwantcake5",
			GracePeriod: 88,
			Offline:     true,
			Claims: jwt.Claims{
				NotBefore: jwt.NewNumericDate(time.Date(2029, 3, 14, 23, 59, 59, 59, time.Local)),
				IssuedAt:  1521485204,
			},
		},

		valid: true,
	},
}

func TestDecodeAndVerify(t *testing.T) {
	for _, entry := range tokenToLicense {
		t.Run(entry.description, func(t *testing.T) {
			RegisterTestingT(t)

			claims, valid := client.DecodeAndVerify(entry.license)
			Expect(valid).Should(Equal(entry.valid), entry.description)

			if entry.valid {
				Expect(claims).Should(Equal(entry.claim), entry.description)
			}
		})
	}
}
