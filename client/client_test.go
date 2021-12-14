package client_test

import (
	"path/filepath"
	"testing"
	"time"

	"os"

	"github.com/davecgh/go-spew/spew"
	. "github.com/onsi/gomega"
	uuid "github.com/satori/go.uuid"
	"github.com/tigera/licensing/client"
	"github.com/tigera/licensing/client/features"
	"gopkg.in/square/go-jose.v2/jwt"

	api "github.com/tigera/api/pkg/apis/projectcalico/v3"
)

var (
	cid1 string

	testPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAxitlRhYjC/RFzL9xDEFkStoysmqxI5Z+ey3bQMVlrHUUSpAO
WsHHBOjhdS7KhsURiMV2W43doQYvtO1lM3/4JuEEbZjFv9AUck5JKG51sTRpU8i/
m79L1w4qCoT3tHx2/JLbvNMTgmbohTwDDKq1Lf7OkG+dADAFecd8TOrDlx4JqmRG
7Pe8yDEAWG/dT84gNh2fvsQZXqCa2LV5fk3Vg/+YRe30dyOMv5oIoBurdIGDs+YP
FQ1vFEPRZ3sAdxIYa2C7efucqKCvEWFcxo43Tfn6j1AJKNqeldEoXwAaHyyy0wZe
u2kYX6zxajBGLMXYpajdOvhHlLPuTB7qVlYDMwIDAQABAoIBAGITihzEyfWZoI3z
1Yw+NNfC48JfgWneipyGFnQY/ff7Pd6lKyWJr+jjJOotDTjkAYiSScCIFr8h46yE
rUhuti7vwJRJPt1uqx/jVNu4x3C7QsGfog0AARXfQblRE5L04qKgQDZUtNwd+Egw
akXzmpW3/R2Iz8gO/DbIHuGmcsSvzqtd80L1ydofWr1Lub8XL7RrLKAwtoHas/3U
G8WIBcvt3l7uTVxpW3e7dFDBVZGimXCTN8fN8sQOR91ZyR/SOJHhsUWQbht0GK1n
XBXlAHzMIZ6cBPT7x366ed0z1xv9ehEJKM4fhtE32i0Z+Zasij43WLkRpjV9C3vj
QJEP1TkCgYEA00X3PjTe6PjxFH0LDDQlQLiLRbfAzhBjuhM/WwGC7yvihJhAVK1o
pC2MxV6pA0zYcX2s2/1HQwSOLPCvuGDU2WRSuiOKCSz6j0m5F5/HuJ068ptzAWg6
NVBMXztTdvCzeMNq7Trm6DlRs5W+5TDK5d90fpps8Q+RKjVXX8dIbY0CgYEA8B9D
3l4YlY12W8Ezv13fQ2ZIAINoPISh+BMFstFnbuuFPA0cjct+DoCN5c/uCHpPwbRu
OFFSbc6+icQTB8vE0PByqukVqwkn2wXH0DwZzKX1u21LqnHgD+gDldTWIH7TF71K
2CL0IrXnWYqhC5yUgGswaHE/jMwQyV6nDSAaI78CgYEAnBOg7jyivFtDxg4GPlK7
fo+Wm79+2Pw0oD8d275HGydBZREQ3T1qA3d++kPO+hgoAdeE/tOidHkGC18XgU9P
jvXVQ5uDmvm2dGpTKYepRNIqvRVnpY95CO+0K9oo88In47wB3xVXhhDqMZAbgTdF
fQJSDkFI3+DPLe5QCPqwn/UCgYEA6ODpqZgIr8Jqr8JItagNCAkCe7z2MvtPOpD4
TdzZO7IfnYX502sv7lCvTdrDOGWnRG4BF42HLAf+sw3+hukRELKiAy/bW+2dQcXx
a/td6iRqlkQBxmR6sfKKx52LrihSAgwLsmLz81YH8ceJOQG65HEQmbp7r8mZ3jJ2
QTyJHXECgYEAttTtALyCCzeS+y3dXxlAi/osOTwexfJIo1IRt7ZNcrcgxhSpn3BW
QMXdVemezHvB3+ZhdjOlzAAvVjB0V/A7ClDkSK1dTm3Wq+3WEQaRgaXMrKFqhvUg
Uii6FtF23mzT1OStrJ++Cl+wLpZ1kb8OuO3B1dETj/S6Ro9JP1OoZiY=
-----END RSA PRIVATE KEY-----`

	testCert = `-----BEGIN CERTIFICATE-----
MIID4DCCAsigAwIBAgIRAP8sPcS0yV6BQiQ6KnHlTyAwDQYJKoZIhvcNAQELBQAw
dTELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNh
biBGcmFuY2lzY28xFDASBgNVBAoTC1RpZ2VyYSBJbmMuMSMwIQYDVQQDExpUaWdl
cmEgSW5jLiBDZXJ0IEF1dGhvcml0eTAeFw0xODAzMjYxNjQ0MDBaFw0xOTAzMjYx
NjQ0MDBaMH0xCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRYwFAYD
VQQHEw1TYW4gRnJhbmNpc2NvMRQwEgYDVQQKEwtUaWdlcmEgSW5jLjErMCkGA1UE
AxMiVGlnZXJhIEluYy4gTGljIHNpZ25pbmcgLSBJbnRlcm5hbDCCASIwDQYJKoZI
hvcNAQEBBQADggEPADCCAQoCggEBAMYrZUYWIwv0Rcy/cQxBZEraMrJqsSOWfnst
20DFZax1FEqQDlrBxwTo4XUuyobFEYjFdluN3aEGL7TtZTN/+CbhBG2Yxb/QFHJO
SShudbE0aVPIv5u/S9cOKgqE97R8dvyS27zTE4Jm6IU8AwyqtS3+zpBvnQAwBXnH
fEzqw5ceCapkRuz3vMgxAFhv3U/OIDYdn77EGV6gmti1eX5N1YP/mEXt9HcjjL+a
CKAbq3SBg7PmDxUNbxRD0Wd7AHcSGGtgu3n7nKigrxFhXMaON035+o9QCSjanpXR
KF8AGh8sstMGXrtpGF+s8WowRizF2KWo3Tr4R5Sz7kwe6lZWAzMCAwEAAaNjMGEw
DgYDVR0PAQH/BAQDAgGGMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0OBBYEFG+np06E
a35Mt31j7V1kP7HMHxstMB8GA1UdIwQYMBaAFGib49CAau909ncUpB/aakasszre
MA0GCSqGSIb3DQEBCwUAA4IBAQA+tiY9jtBR15mS7a6Nz2ppHp0V5qoCClAEyYJG
gCrfZ+TeugPjxVd70N73TYqRRU66sjrX4HLFjyDYTm9H7P5EE84J40C/QpfAeA2n
TiEMgnhCxjnmaxs9JhctoKRENgrIYtTJjnL4WIJN1wuH6HLYClTSUxjS713sI9Jk
V/E7rDjAnTrgg0ZNO20/CgVSNmRgK9cjJPGA0hWAe6XquPal0FAJsfVqwQRIxXGd
TxA0EwUL8NOBHWeVY3sraVHUusOFgMiBr/pdChToD4AJ0jX5a4DzsCRSyMRVz6Dq
pdDGllpqsFfAEMF4hv4A5jhQLPk5Oz5azkoCJ0vcFCtj3nnh
-----END CERTIFICATE-----
`
	testPrivateKeyExpired = `-----BEGIN RSA PRIVATE KEY-----
MIIEowIBAAKCAQEAoofNrVmif8crv+DIvaTvqUqYbGyAlnAURwKPKVVqmqgs813T
pZXHVrAnLuIFWwttti+BRNn8Jjk+YVv8VpfAc2TU4woR3SHD5VRpVK5oF5AFJ4af
Sj3Lou0c+hGJ5gqg03i2l68ba9Xy8qT+n4a+C9t8DdxEXsD4KIeA63qIBOwKCe3B
KK9LEENn0QP75u9pxLVQ4li2ApFN4As615sWAAbsuZ5Pr9wqwphgzHdemxpvB58A
JL41gGDayykRB1ejNdU/MmxXLRbK39PEtmYAYUcF5GqKl4/1FAnaF7mDKqtMthso
zYcv98lEiDpH/E8UXb+OApmaa15+Q3AUmqALrwIDAQABAoIBAHZOtle+DHxIpb75
SAZLviyT4RnjbUKUeR4rbbxfsca8LmREYyCAQ2cFuK/21IEuc4EPWWCd8F5+grrp
82ew9OTKe/B8Tv6RaoBPjpCWl6y3KBladC7dhpKlWNdq1t8900276+XEEAjR5xPb
KIFE2qfU75tDP/1dKAaQhDZkrgguA8jpxW2AbwZrxT7MjxRLWBztyyRgGDFHbHvk
Af/yUsresTuZIAcshHEgj2P9PbHLxA2SlJMuRKGswnRXcXdF//FTWARS0AGU92dk
rD8S9ubPsndD0xCdiMR6F3cmEFXaCTf5krzufn1h/dNMfH8B+SV/Mdo6caj1Xtlo
BdyJ6bECgYEAwVNR560Lnq8JHltT2x54K8O07mgqItXC2Dul/RH026sc8DYaMzth
Oamscd79+72OIPSEzxilITLb0rqvmcX2Qb9UJar57GyNnUHL1xBoGl+j1LAI/UUI
BZTP4MmaQVhW1H4LjGRm7cjDQEqDEKiPv9X5d4vwFpQEQwsyjKzegzcCgYEA1zi2
SvwB96djUc/gFQP51wn610TRawLziAPGEPgNqjqh8A01WK32FauyFkf+wGOr1peB
4PNwyu9T+NLPiuRq0EPKhRECCKzbk/jyV3D3UynuDmCf8GSV3qsMbMVWWFgvb7Ed
iyDfakPtloZ1syxz5SUGwrxsMznude8Ipl5q50kCgYEAp5pFitXyGftjq2a/91qe
EksUJBA4X4T07CQiTplvr7XUW8h7xGi5bJVWBE6v4LzAaH+0WBrkpjiCbVod/PGs
AeoO2K03CSo/R9OQFf6KUjsSPMT0tiZPww71fcsqKXadqJEyD9/HgGSqKaWvpRSN
s2Gdam/ukJR4cWtWwrDoI9cCgYAww3/CM6E6fKmrQr9R46m7CF7WYZhVd8C4A6rf
82QdOtWwLz30Ds5gEJv0InHdI3gu0fsyfdYDlQBgs1sk7CYrdACx762XS5sgxtoZ
59WR+UEf7tKuRAwU/Ip/JqMutyRgWTAJcvRL/oIZhfOrGhpUQ/RpMQoO/URDYlqC
X4g3SQKBgGf2WaIRdMDDXnIew3h7Y3Yz0l+lr7/D4RqzblRlKP157e3BvHUK2Z6+
bI4Uuuod/DMnalWPXoNT6LOQm9F8Sf53hY8wmk2To2HE8Ruy4GS7+LETWukSIYkL
1cvkAD6A5Pmjyxhx8N0mkczU3XFaGc5zSCpHwwO5c1tsoxXL66TZ
-----END RSA PRIVATE KEY-----`

	testCertExpired = `-----BEGIN CERTIFICATE-----
MIID7DCCAtSgAwIBAgIRAJaqj1rVxZWMvOQ7EjJlHSAwDQYJKoZIhvcNAQELBQAw
dTELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNh
biBGcmFuY2lzY28xFDASBgNVBAoTC1RpZ2VyYSBJbmMuMSMwIQYDVQQDExpUaWdl
cmEgSW5jLiBDZXJ0IEF1dGhvcml0eTAeFw0xODA0MDUxODM4MjBaFw0xODA0MDQx
ODM4MTlaMIGIMQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQG
A1UEBxMNU2FuIEZyYW5jaXNjbzEUMBIGA1UEChMLVGlnZXJhIEluYy4xNjA0BgNV
BAMTLVRpZ2VyYSBJbmMuIExpYyBzaWduaW5nIC0gSW50ZXJuYWwgQ0EgZXhwaXJl
ZDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAKKHza1Zon/HK7/gyL2k
76lKmGxsgJZwFEcCjylVapqoLPNd06WVx1awJy7iBVsLbbYvgUTZ/CY5PmFb/FaX
wHNk1OMKEd0hw+VUaVSuaBeQBSeGn0o9y6LtHPoRieYKoNN4tpevG2vV8vKk/p+G
vgvbfA3cRF7A+CiHgOt6iATsCgntwSivSxBDZ9ED++bvacS1UOJYtgKRTeALOteb
FgAG7LmeT6/cKsKYYMx3XpsabwefACS+NYBg2sspEQdXozXVPzJsVy0Wyt/TxLZm
AGFHBeRqipeP9RQJ2he5gyqrTLYbKM2HL/fJRIg6R/xPFF2/jgKZmmtefkNwFJqg
C68CAwEAAaNjMGEwDgYDVR0PAQH/BAQDAgGGMA8GA1UdEwEB/wQFMAMBAf8wHQYD
VR0OBBYEFIgzZdKYca9uSBZ5NvvVxpwO9Ss1MB8GA1UdIwQYMBaAFGib49CAau90
9ncUpB/aakasszreMA0GCSqGSIb3DQEBCwUAA4IBAQCi8dqq/qf9oOWLhs/ZZnqc
Up3MI9CDLkXZ7jmKMN2elH7Y57Ne7m7QLVxLQIP63fU+0LZ0aD7QpL0+YcsS4kuJ
rRsA01JaAry7CRLr2EKahrDeqWTG1T9Wvz/RSQZ7+jDalel7IkH0va7xYsopc1kD
i4Nm6FRWLwSNTcNrfoFtpWzLSkRFejuoA1BiYQdWj3dWyXjSyv2l27hW8w/OGkn3
vYorORYy+3UWQrUUV1wZOf6RG7Dss46O9IkxaiETxuBI4UnaMPOmI6+72StRC5r6
Ej9sQaPis0Lmc16bSSnjrfSsbklMt+rmnIjIXtEJtP7hjj4UMbfsmHFpBHrvfPEc
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
	pkeyPath = "../test-data/pki/intermediate/keys/intermediate.key"

	// Tigera license signing certificate path.
	certPath = "../test-data/pki/intermediate/certs/intermediate.crt"

	absPkeyPath, absCertPath string

	numNodes1 = 555
	numNodes2 = 420
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
			LicenseID:   uuid.NewV4().String(),
			Nodes:       &numNodes2,
			Customer:    "meepster-inc",
			Features:    []string{"nice", "features", "for", "you"},
			GracePeriod: 88,
			Claims: jwt.Claims{
				Expiry:   jwt.NewNumericDate(time.Date(2022, 3, 14, 23, 59, 59, 999999999, time.Local)),
				IssuedAt: jwt.NewNumericDate(time.Now().UTC()),
			},
		},
	},
	{
		description: "only required fields for v2.1 populated",
		claim: client.LicenseClaims{
			LicenseID:   uuid.NewV4().String(),
			Nodes:       &numNodes1,
			Customer:    "cool-cat-inc",
			GracePeriod: 90,
			Claims: jwt.Claims{
				Expiry: jwt.NewNumericDate(time.Date(2022, 3, 14, 23, 59, 59, 999999999, time.Local)),
			},
		},
	},
	{
		description: "partially populated claim",
		claim: client.LicenseClaims{
			LicenseID: uuid.NewV4().String(),
			Nodes:     &numNodes2,
			Customer:  "lame-banana-inc",
			Claims: jwt.Claims{
				Expiry: jwt.NewNumericDate(time.Date(2021, 3, 14, 23, 59, 59, 999999999, time.Local)),
			},
		},
	},
}

// TestGetLicenseFromClaims requires a key / cert that can be decoded with the real public key, e.g. a real
// valid leaf keys. Since those keys are extremely sensitive (if leaked, they can be used to furnish valid licenses)
// we accept their location by env var, and only run these tests if that is specified.
func TestGetLicenseFromClaims(t *testing.T) {
	realKeyPath := os.Getenv("REAL_KEY_PATH")
	realCertPath := os.Getenv("REAL_CERT_PATH")
	if realKeyPath == "" || realCertPath == "" {
		t.Skip("REAL_KEY_PATH / REAL_CERT_PATH not specified: skipping decode tests")
	}

	for _, entry := range claimToJWTTable {
		t.Run(entry.description, func(t *testing.T) {
			RegisterTestingT(t)

			lic, err := client.GenerateLicenseFromClaims(entry.claim, realKeyPath, realCertPath)

			spew.Dump(time.Now().Local())

			// We cannot assert the token because it's hard to generate the exact random feed used to encrypt the JWT.
			Expect(err).NotTo(HaveOccurred(), entry.description)

			// We can verify the generated resource's Objectmeta name.
			Expect(lic.Name).Should(Equal("default"), entry.description)

			claims, err := client.Decode(*lic)
			Expect(err).NotTo(HaveOccurred(), entry.description)
			Expect(claims).Should(Equal(entry.claim), entry.description)
		})
	}
}

var tokenToLicense = []struct {
	description string
	license     api.LicenseKey
	claim       client.LicenseClaims
	corrupt     bool
}{
	//{
	//	description: "fully populated uncorrupt claim",
	//	license: api.LicenseKey{
	//		ObjectMeta: v1.ObjectMeta{
	//			Name: "default",
	//		},
	//		Spec: api.LicenseKeySpec{
	//			Token:       "eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJVWHdPUDdKa3RuOVhXRTMzIiwidGFnIjoiaEdWUk9FQW9GcmluNEN5V2ZyTFVfQSIsInR5cCI6IkpXVCJ9.fNrEWcFbBh1UxOvxQOmIzA.wG0JWHnG_Suc4APp.6zu2Uu4Sm2BjJSfU9F8FsJzYj7jz5Qs4tK0lG0X_hr1lro2KFa2QEKZ4iRHcrcp3MFvQjp8VV1LYjwVqzfwqVfKjxwBZxbtUvDtDbGz3p7UmlhHSGnHyW2O_CKbf1q-UWWsAU9HNKkKKSzPIuIjXSWs6YfaBhISqJ42dbJK4ORM_Me6DXvuP3FmxEvulKSUjn0g4iUmID159svJppryyebyiVwddY1-SHZmqzPPnh0X2FTv_H1gSPhInksCdZFbnIPNUFt1Y9ZSR1xlwm-tM4sISiIxhYhLbV3zRb4_o--XUZTbiSVMCiCL8gjwDSyx80APW6Hv4Fsa3wlML0tlSVvOunNQ46k2NIXfE1GXvXp4r47TgEnq5B_peasrldKL6RSILtkU0j-iIpnd-5avy_yh-Vv-Al7q548frudKilbcBE2JmXmdGTv4zXUMIgv-tzPjrnw5dYjcoYhJrQNX04UPXVMytP3gWkg1g1s30iVQi-4WowogUJNj-NzbYHfi32WYjYmFJ4XHAgcIc1Ji-RoyJSKcjEu2VlFKRzkOhf8ADGY9xLNfHtLLEEq8tlgo5dYa-MD0vd249P5bXp9ePBbh_WXBAiGeIjj26hxFJ0R1cYhG8PFZiMxrnJR2p3aHtVxQuH-scWq65Gagm_asHitgLd88CC2fa5JYuNFjCKYWcBk96NIi545mT7SaIOptcmh19CjPweZi5kAHK0NT2dkqY54wu0XQEtJj66DPSp4muU9p-fFbNK7NrfIMMuPUXJhUaLTebGCfWUzRG02KfIezVfTteB9dkByJx44579uhUmd6sd6kDNE3yAVXf7mBr2w7w-NVxgu-E64G9r-HBC5Z48iJp6zqqVyTBGKvuIzMlMbLX_J8KTGU--JE.F1Cq9fv-6aiOvGUidHaegQ",
	//			Certificate: testCertExpired,
	//		},
	//	},
	//
	//	claim: client.LicenseClaims{
	//		LicenseID:   "5fa38831-fca5-4ea1-9722-ac601aa6852a",
	//		Nodes:       &numNodes2,
	//		Customer:    "meepster-inc",
	//		Features:    []string{"cnx", "all"},
	//		GracePeriod: 88,
	//		Claims: jwt.Claims{
	//			Expiry:   jwt.NewNumericDate(time.Date(2022, 3, 14, 23, 59, 59, 59, time.Local)),
	//			IssuedAt: 1521764255,
	//		},
	//	},
	//
	//	corrupt: false,
	//},
	//{
	//	description: "claim with the JWT header meddled with",
	//	license: api.LicenseKey{
	//		ObjectMeta: v1.ObjectMeta{
	//			Name: "default",
	//		},
	//		Spec: api.LicenseKeySpec{
	//			Token:       "eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJ3WWpuYjV6TTF5MlV6RFZ4IiwidFnIjoiN1dSUkxPanNGQ0F1R3pNRGg5akc1USIsInR5cCI6IkpXVCJ9.HtXrz5-Q_vVfKwgn9Ig_zQ.xf6FZYH3315Tffzv.v7JNl7qOWTivF3Y0Fla-5uG-SM7zCVWcOWEncS7y5kc_uIIRTvTqXV7LAB0b6rZFkXGYxo3X0nBADh7yVJO2S9LX3AbjhF4g_5Vu1uVHwNyKEmSxoMhJGK8v0kwtmXWF7dgICKlAWcSE2kscr-1P-m-MgjTPIZaQU27EN3KFNBgPtLalSKcTRoKMWbqnZRyZFB4gIhpXRKOi2wSlRwbzflumRt5PBGQ6AAdqJaZhEDKYIRVwiYiLh8ODXC2WNhF9KS7GqXRE9QopOcQkh3n_AAADIgzOMdrVr26VTXKXZlwtTYZ5cNPxRZA7QkQVB9HMh7WwwstcSLlVRnHcGZJwmTUfpdGExAywCu4DkqJRnarfJUmG1Y86ecOFnmuycFo0NPuruUEXUG33Nd_670qOWzICjqu68cx3AXcwh46m8hZGR3Zbs1usYfrWTVfFZxNUYlAOCmjrnIAKfxDe4B4fBKYEyFM7PTUQj1UTChgv5G3wRBZiVPDv67gnOrqtQQNyAtJvWsaSdxEu5LGzO68ntauYM4wohnqx4JBzFrd5YkWivHf10yFb7_mGYxhqG7_lPiWAd7zxJNGYrOHi8qEMPFtKANI4UKLAbyXVgPJuTo_kAmoHpSqvAf2DTNODBJQb_hl6F6gX0gWsJIQ1V7O7xn6aAc0nkiizYSLuoKLSsF8rWSyASnPuHhc5AeFVEqA8oRYeZLMh9BBYr8w3kGa6eobtp8j8g2YcEy-KSCgxuef94OIRn6EPbvkfhhz8bZm9c1670N701J91WnIG7l1WXFAxXnfO055W0ulpbE99sw.HACGOFtKA6ZvoAg4Prgiaw",
	//			Certificate: testCert,
	//		},
	//	},
	//
	//	claim: client.LicenseClaims{},
	//
	//	corrupt: true,
	//},
	//{
	//	description: "claim with the JWT payload meddled with",
	//	license: api.LicenseKey{
	//		ObjectMeta: v1.ObjectMeta{
	//			Name: "default",
	//		},
	//		Spec: api.LicenseKeySpec{
	//			Token:       "eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJ3WWpuYjV6TTF5MlV6RFZ4IiwidGFnIjoiN1dSUkxPanNGQ0F1R3pNRGg5akc1USIsInR5cCI6IkpXVCJ9.HtXrz5-Q_vVfKwgn9Ig_zQ.xf6FZYH3315Tffzv.v7JNl7qOWTivFY0Fla-5uG-SM7zCVWcOWEncS7y5kc_uIIRTvTqXV7LAB0b6rZFkXGYxo3X0nBADh7yVJO2S9LX3AbjhF4g_5Vu1uVHwNyKEmSxoMhJGK8v0kwtmXWF7dgICKlAWcSE2kscr-1P-m-MgjTPIZaQU27EN3KFNBgPtLalSKcTRoKMWbqnZRyZFB4gIhpXRKOi2wSlRwbzflumRt5PBGQ6AAdqJaZhEDKYIRVwiYiLh8ODXC2WNhF9KS7GqXRE9QopOcQkh3n_AAADIgzOMdrVr26VTXKXZlwtTYZ5cNPxRZA7QkQVB9HMh7WwwstcSLlVRnHcGZJwmTUfpdGExAywCu4DkqJRnarfJUmG1Y86ecOFnmuycFo0NPuruUEXUG33Nd_670qOWzICjqu68cx3AXcwh46m8hZGR3Zbs1usYfrWTVfFZxNUYlAOCmjrnIAKfxDe4B4fBKYEyFM7PTUQj1UTChgv5G3wRBZiVPDv67gnOrqtQQNyAtJvWsaSdxEu5LGzO68ntauYM4wohnqx4JBzFrd5YkWivHf10yFb7_mGYxhqG7_lPiWAd7zxJNGYrOHi8qEMPFtKANI4UKLAbyXVgPJuTo_kAmoHpSqvAf2DTNODBJQb_hl6F6gX0gWsJIQ1V7O7xn6aAc0nkiizYSLuoKLSsF8rWSyASnPuHhc5AeFVEqA8oRYeZLMh9BBYr8w3kGa6eobtp8j8g2YcEy-KSCgxuef94OIRn6EPbvkfhhz8bZm9c1670N701J91WnIG7l1WXFAxXnfO055W0ulpbE99sw.HACGOFtKA6ZvoAg4Prgiaw",
	//			Certificate: testCert,
	//		},
	//	},
	//
	//	claim: client.LicenseClaims{},
	//
	//	corrupt: true,
	//},
	//{
	//	description: "claim with the JWT signed by some evil random private key",
	//	license: api.LicenseKey{
	//		ObjectMeta: v1.ObjectMeta{
	//			Name: "default",
	//		},
	//		Spec: api.LicenseKeySpec{
	//			Token:       "eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJVeG1hUnBucS1Oc2JORWY1IiwidGFnIjoiZkFnR2I0U2R5WlRTTWJVTFZSVE91dyIsInR5cCI6IkpXVCJ9.ao8K2OAgme4kwVejNn-Lvg.CyEws8QbrDGjtkVx.Pebt9PmCpWvPcVYkzkY2BSP92RGCOfg7oGHSfo5MiabnXXn6KDQ6rT2wxHjcHTNcszYO8nZ_w4nUIvH0Vg-7VAbHhvYFpsbtuc8eXRSbqV9Vt0-jm4N9iQFbT5bEi-qyPk5p-OjK_UO8tAPll7foQz9DlqG1h55Pn2RyrjL2-oTJeDb5b7uRkLFASeD-ApqB6NylQ6oskCr9GN5vHaV5_tRaoaWTlCPFwUIQc1TMwoBDoyNTWJUV45QeuT6ha1T4IgiDS7uJcvPb7omm7dhoXK5aw-b-G8wVlWbfD-0ygzPr9qehkh9IYmJAQtYo46dTJBKIInQUss-IpURNUQKVuYrODFkw4GEpQ4FQAamIktYt_EHudzMrrtJM3xhvtYT9bYJz-0_wYnloy7kJMd7JHPaRxH3wICAw0UUe-0F8sViA5NTnADKSXnpWRRDArsFKezywdUqCgRV9lwHbaDKSJFaMSOMJ3BmTXOz_vJ1hiWCjelAUU0sE6r0tcIYPgc705hLYnRb5Xk_qePhtFdAZkqRkymnYJVRRYmQhVYaDEB33E9UYFLqL1EOhkfRnu-iNuMky9OfjuwrjoBaVJDlBQ9y76iOMoDZr4hpEIsESli8nY0MzzHLc2T4WUd1rx9XSw7VaojSYPvpK9JWhJkWcQVb28FNJB6Fui7V_T1bnF44vBqy2OKY3iK-OotULdm76Jm_rSXgpoJldUOjc31f6qTD78SeZ5UhyxgLGCzS5lHri1FCiYDjy6dcFGNfoWJ1Lpj5mTY_4OLnfLG2yqlyqRfrX8bTq5X0.V1LdXb0VgrJDlkeQ95GWmQ",
	//			Certificate: testCert,
	//		},
	//	},
	//
	//	claim: client.LicenseClaims{},
	//
	//	corrupt: true,
	//},
	//{
	//	description: "claim with the JWT signed by tigera but certificate is swapped out with an evil certificate",
	//	license: api.LicenseKey{
	//		ObjectMeta: v1.ObjectMeta{
	//			Name: "default",
	//		},
	//		Spec: api.LicenseKeySpec{
	//			Token:       "eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJ3WWpuYjV6TTF5MlV6RFZ4IiwidGFnIjoiN1dSUkxPanNGQ0F1R3pNRGg5akc1USIsInR5cCI6IkpXVCJ9.HtXrz5-Q_vVfKwgn9Ig_zQ.xf6FZYH3315Tffzv.v7JNl7qOWTivF3Y0Fla-5uG-SM7zCVWcOWEncS7y5kc_uIIRTvTqXV7LAB0b6rZFkXGYxo3X0nBADh7yVJO2S9LX3AbjhF4g_5Vu1uVHwNyKEmSxoMhJGK8v0kwtmXWF7dgICKlAWcSE2kscr-1P-m-MgjTPIZaQU27EN3KFNBgPtLalSKcTRoKMWbqnZRyZFB4gIhpXRKOi2wSlRwbzflumRt5PBGQ6AAdqJaZhEDKYIRVwiYiLh8ODXC2WNhF9KS7GqXRE9QopOcQkh3n_AAADIgzOMdrVr26VTXKXZlwtTYZ5cNPxRZA7QkQVB9HMh7WwwstcSLlVRnHcGZJwmTUfpdGExAywCu4DkqJRnarfJUmG1Y86ecOFnmuycFo0NPuruUEXUG33Nd_670qOWzICjqu68cx3AXcwh46m8hZGR3Zbs1usYfrWTVfFZxNUYlAOCmjrnIAKfxDe4B4fBKYEyFM7PTUQj1UTChgv5G3wRBZiVPDv67gnOrqtQQNyAtJvWsaSdxEu5LGzO68ntauYM4wohnqx4JBzFrd5YkWivHf10yFb7_mGYxhqG7_lPiWAd7zxJNGYrOHi8qEMPFtKANI4UKLAbyXVgPJuTo_kAmoHpSqvAf2DTNODBJQb_hl6F6gX0gWsJIQ1V7O7xn6aAc0nkiizYSLuoKLSsF8rWSyASnPuHhc5AeFVEqA8oRYeZLMh9BBYr8w3kGa6eobtp8j8g2YcEy-KSCgxuef94OIRn6EPbvkfhhz8bZm9c1670N701J91WnIG7l1WXFAxXnfO055W0ulpbE99sw.HACGOFtKA6ZvoAg4Prgiaw",
	//			Certificate: evilCert,
	//		},
	//	},
	//
	//	claim: client.LicenseClaims{},
	//
	//	corrupt: true,
	//},
	//{
	//	description: "claim with the JWT signed by an evil private key but certificate is still the tigera original cert",
	//	license: api.LicenseKey{
	//		ObjectMeta: v1.ObjectMeta{
	//			Name: "default",
	//		},
	//		Spec: api.LicenseKeySpec{
	//			Token:       "eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJLeWE0VHpEaWY2eFM3TTl2IiwidGFnIjoiYVhHR3d0alczSjhKeWgtb2hWajRJdyIsInR5cCI6IkpXVCJ9.LgbBH-IGmLH2iUFY171xwA.NImD2DVyH1ahbruT.DHhdADLX7BfwwYoknoTnPEQGh7vItF7YhYukfPDm_VlwgERXTDdqb6wFQQOZOvFFlcMRYBBzDQBguSkYEHYWegHIuZ7Amfh8uCcI0l93BPz1TrOZdX4fukikb5YVTbRJjxgJTvakucG9dh45hwks9gUCGdXFvVAJH_wMDc_kPVeb0fx84f_H30gNswvKItyIT09lOiRCfy9HOGdpo1RlA0UCZvIPYD9zSl1_ldGZ5Oj2RYz9HU7bhuqV4AU7OuglE_8yvNMmkqSD9BmiLOxzxMVvg3uj5trmuTOy4pAZuchykM3p-DgGiWuo4kyaHvpcfIISSyBU8xtVMyWALayeaschyvlAvRJHAVjKd9Cubx5akA23w4KpBGsJ2EgQPNmyHdEoxqKohO6KbYcOvsD7PThH8e9UV7GgGrQp4OUBZXfym-_yi_erI6FC91n3rgcSMqYpIrhC5-dPSExKuPVA_94dlcP-cDxAtuL8W0T8mafTqKl4Vg-Ojaj7pul4-i7223loZSbkYEpuoTzHYglgB2_PfHgkZsqgl8adlm7muKpxSe_TH-6wQh6fXxGzUJEu7DLvcy82r5v_HcWtJUj43qu8BTHR4sc4_1NU8eHya_HtwgvOo98Ze1Gd9qC_GOFkMYomEk2ogarPnGGKD-gfMN3GxziUz5d4kpb8mzknGIX5hqaxcslV4HDnSA97zjssyajg1Eh-a6xOIaPOlYW3YzXQ3GQPABLn18V2hFCNhB-ml6KWceYA6EsxnKqdEK2KN8dnDGESdjwCIUfcY7KFRD30qhAOUAKpU14.YvpAmE0JPK1Brn7kgGphlg",
	//			Certificate: testCert,
	//		},
	//	},
	//
	//	claim: client.LicenseClaims{},
	//
	//	corrupt: true,
	//},
	//{
	//	// TODO (gunjan5): THIS TEST SHOULD FAIL ONCE WE ADD CERT CHAIN VALIDATION!!!!
	//	description: "claim with the JWT signed by an evil private key with an evil cert",
	//	license: api.LicenseKey{
	//		ObjectMeta: v1.ObjectMeta{
	//			Name: "default",
	//		},
	//		Spec: api.LicenseKeySpec{
	//			Token:       "eyJhbGciOiJBMTI4R0NNS1ciLCJjdHkiOiJKV1QiLCJlbmMiOiJBMTI4R0NNIiwiaXYiOiJyS0UycFlocnhkWl9Fd2NOIiwidGFnIjoiQ0ZEcUotaTJTRUQwVmN5Si1EdGxiUSIsInR5cCI6IkpXVCJ9.FvqJP0N6R-udVr993WXaow.T8D4LnRW5LHtlI9N.NXK2hpRK9jHK6g00PJfGD1xK5YDDtSYP0kMM6BCEjSNcGiCrKAt9bDEHsYUttkY74OO_pMfGJOx_-RdFcfk_JxKJLR2mtTX6Tyx0oP3QN6OoHbzqfIEs_FWjqyLvxnGvIgzJpusF_LBg3MOuRflLr8Wn9bGNZN37er0PtZs2L5KqtgFmPKe-IqVNXIqZ7F1DUhwNmWruGguffGtnavuXcHYvqyAX5PUsatia0tGrkIP8810DgwqPBqzZquZIncR6n_1HdW1jFFJ4SWv2J4CkKct74jxPbxQHoItvTeqtlvntjclri1LiLRzkbPU4yYA3MFibexaIbn6yD6aCZkPOwjtkciB7f-Wg-Vx7DHV6_XbEtTFjummiJY87e8R-gCxFQNRmZ5zJKuoFCo_KGLL_HRG6plNmt3M6Z3vrk28Gx26Pv_caSA9IY3hcGn89ah4Nif1pSf7ioRZDjeac3wusBper_TsZ5FJd-DSI91laYNwAh3_Obp0YswxigFLIpZzGICac6CPFx58zQp8XZAPG8LeL049Byx_yTheOwfsWIeplrBrnCCXqSQ4fPrW2Lx7aS-VyWgcDX7JhO54YzGsL9k5WUcYjVxCsO1tPdfv4uzVBRJYR27oodwdCs0cOEAP8uZBDpGGFeVlWDAZatSCX8MLSBzu3Fo97HafabQ8jg3Piy0XTaBUC1fWJU6ygLdLxtzpRERUJL32-DbdWw0j4YfgDrqYZhpk_XXhNHiPKUbyC7kPh8jaFwHgYbq2jwHBMo4pOs3tLH9-36q4FNeHOIFN7ZqsGENLl-3bgHfRj5eJT1nhcc2z_6D0036pgZDcTOh_wfFoI0FujD0A3NKNhBUueo_rTjFIqA7l_WiNZj0HLaCU1ezx1GuoM.UPz5crKaIBcSwsaeLvaq6w",
	//			Certificate: evilCert,
	//		},
	//	},
	//
	//	claim: client.LicenseClaims{
	//		LicenseID:   "a34d87c2-aea0-4b40-8c8b-1dae3fd13990",
	//		Nodes:       &numNodes1,
	//		Customer:    "iwantcake5",
	//		Features: []string{"cnx", "all"},
	//		GracePeriod: 88,
	//		Claims: jwt.Claims{
	//			Expiry:   jwt.NewNumericDate(time.Date(2029, 3, 14, 23, 59, 59, 59, time.Local)),
	//			IssuedAt: 1521765193,
	//		},
	//	},
	//
	//	corrupt: false,
	//},
}

func TestDecodeAndVerify(t *testing.T) {
	for _, entry := range tokenToLicense {
		t.Run(entry.description, func(t *testing.T) {
			RegisterTestingT(t)

			//lic, err := client.GenerateLicenseFromClaims(entry.claim, absPkeyPath, absCertPath)
			//spew.Dump(lic)

			claims, err := client.Decode(entry.license)

			if entry.corrupt {
				Expect(err).To(HaveOccurred(), entry.description)
			} else {
				Expect(err).NotTo(HaveOccurred(), entry.description)
				Expect(claims).Should(Equal(entry.claim), entry.description)
			}
		})
	}
}

func keys(set map[string]bool) []string {
	var keys []string
	for k, _ := range set {
		keys = append(keys, k)
	}

	return keys
}
func TestFeatureFlags(t *testing.T) {
	numNodes := 5
	sampleClaims := client.LicenseClaims{
		LicenseID:   "yaddayadda",
		Nodes:       &numNodes,
		Customer:    "MyFavCustomer99",
		GracePeriod: 90,
		Claims: jwt.Claims{
			Expiry: jwt.NumericDate(time.Now().Add(72 * time.Hour).UTC().Unix()),
			Issuer: "Gunjan's office number 5",
		},
	}

	t.Run("a license with 'all' features states that each feature is enabled.", func(t *testing.T) {
		RegisterTestingT(t)

		claims := sampleClaims
		claims.Features = []string{features.All}
		Expect(claims.ValidateFeature(features.AWSCloudwatchMetrics)).To(BeTrue())
	})

	t.Run("a license only valid for cloudwatch metrics is valid for cloudwatch metrics.", func(t *testing.T) {
		RegisterTestingT(t)

		claims := sampleClaims
		claims.Features = []string{features.AWSCloudwatchMetrics}
		Expect(claims.ValidateFeature(features.AWSCloudwatchMetrics)).To(BeTrue())
	})

	t.Run("a license only valid for cloudwatch metrics is not valid for ipsec.", func(t *testing.T) {
		RegisterTestingT(t)

		claims := sampleClaims
		claims.Features = []string{features.AWSCloudwatchMetrics}
		Expect(claims.ValidateFeature(features.IPSec)).To(BeFalse())
	})

	t.Run("a license with 'cnx|all' features states that each feature is enabled.", func(t *testing.T) {
		RegisterTestingT(t)

		claims := sampleClaims
		claims.Features = []string{"cnx", features.All}

		for f := range features.EnterpriseFeatures {
			Expect(claims.ValidateFeature(f)).To(BeTrue())
		}
	})

	t.Run("a license with 'cloud|community' package states any cloud community feature is enabled.", func(t *testing.T) {
		RegisterTestingT(t)

		claims := sampleClaims
		claims.Features = append([]string{"cloud", "community"}, keys(features.CloudCommunityFeatures)...)

		for f := range features.CloudCommunityFeatures {
			Expect(claims.ValidateFeature(f)).To(BeTrue())
		}
	})

	t.Run("a license with 'cloud|starter' package states any cloud starter feature is enabled.", func(t *testing.T) {
		RegisterTestingT(t)

		claims := sampleClaims
		claims.Features = append([]string{"cloud", "starter"}, keys(features.CloudStarterFeatures)...)

		for f := range features.CloudStarterFeatures {
			Expect(claims.ValidateFeature(f)).To(BeTrue())
		}
	})

	t.Run("a license with 'cloud|pro' package states any cloud pro feature is enabled.", func(t *testing.T) {
		RegisterTestingT(t)

		claims := sampleClaims
		claims.Features = append([]string{"cloud", "pro"}, keys(features.CloudProFeatures)...)

		for f := range features.CloudProFeatures {
			Expect(claims.ValidateFeature(f)).To(BeTrue())
		}
	})
}

func TestLicenseStatus(t *testing.T) {
	t.Run("empty claims status is none", func(t *testing.T) {
		RegisterTestingT(t)

		var claims *client.LicenseClaims
		Expect(claims.Validate()).To(Equal(client.NoLicenseLoaded))
	})

	t.Run("valid claims are valid", func(t *testing.T) {
		RegisterTestingT(t)

		claims := client.LicenseClaims{
			GracePeriod: 0,
			Claims: jwt.Claims{
				Expiry: jwt.NumericDate(time.Now().Add(72 * time.Hour).UTC().Unix()),
			},
		}

		Expect(claims.Validate()).To(Equal(client.Valid))
	})

	t.Run("grace period claims are in grace period", func(t *testing.T) {
		RegisterTestingT(t)

		claims := client.LicenseClaims{
			GracePeriod: 2,
			Claims: jwt.Claims{
				Expiry: jwt.NumericDate(time.Now().UTC().Unix()),
			},
		}

		Expect(claims.Validate()).To(Equal(client.InGracePeriod))
	})

	t.Run("expired claims are expired", func(t *testing.T) {
		RegisterTestingT(t)

		claims := client.LicenseClaims{
			GracePeriod: 0,
			Claims: jwt.Claims{
				Expiry: jwt.NumericDate(time.Now().UTC().Unix()),
			},
		}

		Expect(claims.Validate()).To(Equal(client.Expired))
	})

	t.Run("expired claims are expired after grace period", func(t *testing.T) {
		RegisterTestingT(t)

		claims := client.LicenseClaims{
			GracePeriod: 1,
			Claims: jwt.Claims{
				Expiry: jwt.NumericDate(time.Now().Add(-48 * time.Hour).UTC().Unix()),
			},
		}

		Expect(claims.Validate()).To(Equal(client.Expired))
	})

}
