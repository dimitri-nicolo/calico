package client_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"path/filepath"
	"time"

	"github.com/tigera/licensing/client"
	"gopkg.in/square/go-jose.v2/jwt"
	"github.com/davecgh/go-spew/spew"
	"github.com/satori/go.uuid"
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

	// Tigera private key location.
	pkeyPath = "./test_tigera.io_private_key.pem"

	// Tigera license signing certificate path.
	certPath = "./test_tigera.io_certificate.pem"

	absPkeyPath, absCertPath string
)

func init() {
	absPkeyPath, _ = filepath.Abs(pkeyPath)
	absCertPath, _ = filepath.Abs(certPath)
}

func TestCryptoBasics(t *testing.T) {
	// Generate Pub/Priv key pair.
	//priv, err := cryptolicensing.GenerateKeyPair()
	//if err != nil {
	//	log.Fatalf("error generating pub/priv key pair")
	//}
	//
	//err = cryptolicensing.SavePrivateKeyAsPEM(priv, "privateKey.pem")
	//if err != nil {
	//	log.Fatalf("error saving private key to file: %s", err)
	//}
	//
	//// Generate x.509 certificate.
	//now := time.Now()
	//// Valid for one year from now.
	//then := now.Add(60 * 60 * 24 * 365 * 1000 * 1000 * 1000)
	//derBytes, err := cryptolicensing.Generatex509Cert(now, then, priv)
	//if err != nil {
	//	log.Fatalf("error generating x.509 certificate: %s", err)
	//}
	//
	//err = cryptolicensing.SaveCertToFile(derBytes, "tigera.io.cer")
	//if err != nil {
	//	log.Fatalf("error saving cert to file: %s", err)
	//}
	//
	//err = cryptolicensing.SaveCertAsPEM(derBytes, "tigera.io.pem")
	//if err != nil {
	//	log.Fatalf("error saving cert to file: %s", err)
	//}
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
				IssuedAt: jwt.NewNumericDate(time.Now().Local()),
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
			CustomerID:  uuid.NewV4().String(),
			Nodes:       1000,
			Name:        "lame-banana-inc",
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

//var tokenToLicense = []struct {
//	description string
//	license     api.LicenseKey
//	claim       client.LicenseClaims
//}{
//	{
//		description: "fully populated claim",
//		license: api.LicenseKey{
//			ObjectMeta: v1.ObjectMeta{
//				Name: "default",
//			},
//			Spec: api.LicenseKeySpec{
//				Token:    "eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJ3WWpuYjV6TTF5MlV6RFZ4IiwidGFnIjoiN1dSUkxPanNGQ0F1R3pNRGg5akc1USIsInR5cCI6IkpXVCJ9.HtXrz5-Q_vVfKwgn9Ig_zQ.xf6FZYH3315Tffzv.v7JNl7qOWTivF3Y0Fla-5uG-SM7zCVWcOWEncS7y5kc_uIIRTvTqXV7LAB0b6rZFkXGYxo3X0nBADh7yVJO2S9LX3AbjhF4g_5Vu1uVHwNyKEmSxoMhJGK8v0kwtmXWF7dgICKlAWcSE2kscr-1P-m-MgjTPIZaQU27EN3KFNBgPtLalSKcTRoKMWbqnZRyZFB4gIhpXRKOi2wSlRwbzflumRt5PBGQ6AAdqJaZhEDKYIRVwiYiLh8ODXC2WNhF9KS7GqXRE9QopOcQkh3n_AAADIgzOMdrVr26VTXKXZlwtTYZ5cNPxRZA7QkQVB9HMh7WwwstcSLlVRnHcGZJwmTUfpdGExAywCu4DkqJRnarfJUmG1Y86ecOFnmuycFo0NPuruUEXUG33Nd_670qOWzICjqu68cx3AXcwh46m8hZGR3Zbs1usYfrWTVfFZxNUYlAOCmjrnIAKfxDe4B4fBKYEyFM7PTUQj1UTChgv5G3wRBZiVPDv67gnOrqtQQNyAtJvWsaSdxEu5LGzO68ntauYM4wohnqx4JBzFrd5YkWivHf10yFb7_mGYxhqG7_lPiWAd7zxJNGYrOHi8qEMPFtKANI4UKLAbyXVgPJuTo_kAmoHpSqvAf2DTNODBJQb_hl6F6gX0gWsJIQ1V7O7xn6aAc0nkiizYSLuoKLSsF8rWSyASnPuHhc5AeFVEqA8oRYeZLMh9BBYr8w3kGa6eobtp8j8g2YcEy-KSCgxuef94OIRn6EPbvkfhhz8bZm9c1670N701J91WnIG7l1WXFAxXnfO055W0ulpbE99sw.HACGOFtKA6ZvoAg4Prgiaw",
//				Certificate: testCert,
//			},
//		},
//
//		claim: client.LicenseClaims{
//			CustomerID:  "meow23424coldcovfefe0nmyfac3",
//			Nodes:       420,
//			Name:        "meepster-inc",
//			Features:    []string{"nice", "features", "for", "you"},
//			GracePeriod: 88,
//			Offline:     true,
//			Claims: jwt.Claims{
//				NotBefore: jwt.NewNumericDate(time.Date(2020, 3, 14, 23, 59, 59, 59, time.Local)),
//			},
//		},
//	},
//}

//func TestDecodeAndVerify(t *testing.T) {
//	for _, entry := range tokenToLicense {
//		t.Run(entry.description, func(t *testing.T) {
//			RegisterTestingT(t)
//
//			claims, valid := client.DecodeAndVerify(entry.license)
//			Expect(valid).To(BeTrue())
//
//			Expect(claims).Should(Equal(entry.claim))
//		})
//	}
//}
