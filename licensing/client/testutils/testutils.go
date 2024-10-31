// Package testutils provides a test decode method which can be used with certs not signed by the official
// tigera root key pair. This is useful for testing license generation without needing to pass around the real certs.
// But it should not be used for anything else! Do not import this into production code under any circumstance!!
package testutils

import (
	"fmt"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	api "github.com/tigera/api/pkg/apis/projectcalico/v3"

	"github.com/projectcalico/calico/licensing/client"
	cryptolicensing "github.com/projectcalico/calico/licensing/crypto"
)

const IntermediateCert = `-----BEGIN CERTIFICATE-----
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
-----END CERTIFICATE-----`

const PrivateKey = `-----BEGIN RSA PRIVATE KEY-----
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

const RootCert = `-----BEGIN CERTIFICATE-----
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

// IsDevLicense is an error returned when using the dev license decoder. It is used to prevent
// clients from accidentally using the dev decoder.
type IsDevLicense struct {
	s string
}

func (e *IsDevLicense) Error() string {
	return e.s
}

// DecodeDevLicense takes a license resource and decodes the claims using the dev root cert.
//
// DO NOT USE THIS IN ANY PRODUCTION CODE SHIPPED IN ANY BINARIES!!
//
// This Method is for testing use only. It returns the decoded client.LicenseClaims and an error.
// To prevent accidental use of this function, the error is _always_ non-nil. If the function succeeded,
// it will be an IsDevLicense error. Any other error means the license is corrupted.
func DecodeDevLicense(lic api.LicenseKey) (client.LicenseClaims, error) {
	tok, err := jwt.ParseSignedAndEncrypted(
		lic.Spec.Token,
		[]jose.KeyAlgorithm{jose.A128GCMKW},
		[]jose.ContentEncryption{jose.A128GCM},
		[]jose.SignatureAlgorithm{jose.PS512})
	if err != nil {
		return client.LicenseClaims{}, fmt.Errorf("error parsing license: %s", err)
	}

	nested, err := tok.Decrypt([]byte("i༒2ஹ阳0?!pᄚ3-)0$߷५ૠm"))
	if err != nil {
		return client.LicenseClaims{}, fmt.Errorf("error decrypting license: %s", err)
	}

	cert, err := cryptolicensing.LoadCertFromPEM([]byte(lic.Spec.Certificate))
	if err != nil {
		return client.LicenseClaims{}, fmt.Errorf("error loading license certificate: %s", err)
	}

	rootCert, err := cryptolicensing.LoadCertFromPEM([]byte(RootCert))
	if err != nil {
		return client.LicenseClaims{}, fmt.Errorf("error loading license certificate: %s", err)
	}

	// Check if the certificate was signed by the provided root certificate.
	if err = cert.CheckSignatureFrom(rootCert); err != nil {
		return client.LicenseClaims{}, fmt.Errorf("failed to verify the certificate: %s", err)
	}

	var claims client.LicenseClaims
	if err := nested.Claims(cert.PublicKey, &claims); err != nil {
		return client.LicenseClaims{}, fmt.Errorf("error parsing license claims: %s", err)
	}

	// Return the succesfully decoded claim and an IsDevLicense error.
	return claims, &IsDevLicense{"This license is for development only!"}
}
