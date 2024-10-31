package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"time"

	cryptolicensing "github.com/projectcalico/calico/licensing/crypto"
)

// Verifying with a custom list of root certificates.

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

const certPEM = `-----BEGIN CERTIFICATE-----
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

func main() {
	//// Generate Pub/Priv key pair.
	//priv, err := cryptolicensing.GenerateKeyPair()
	//if err != nil {
	//	log.Fatalf("error generating pub/priv key pair")
	//}
	//
	//err = cryptolicensing.SavePrivateKeyAsPEM(priv, "test_evil_private_key.pem")
	//if err != nil {
	//	log.Fatalf("error saving private key to file: %s", err)
	//}
	//
	//// Generate x.509 certificate.
	//now := time.Now()
	//// Valid for one year from now.
	//then := now.Add(60 * 60 * 24 * 365 * 1000 * 1000 * 1000 * 5)
	//derBytes, err := cryptolicensing.Generatex509Cert(now, then, priv)
	//if err != nil {
	//	log.Fatalf("error generating x.509 certificate: %s", err)
	//}
	//
	////err = cryptolicensing.SaveCertToFile(derBytes, "test_evil_cert.cer")
	////if err != nil {
	////	log.Fatalf("error saving cert to file: %s", err)
	////}
	//
	//err = cryptolicensing.SaveCertAsPEM(derBytes, "test_evil_cert.pem")
	//if err != nil {
	//	log.Fatalf("error saving cert to file: %s", err)
	//}
	//

	//CreateCertKeyPair("./rootSSCert.pem", "./rootPKey.pem", 10)
	//
	//// Generate x.509 certificate.
	//now := time.Now()
	//// Valid for one year from now.
	//then := now.Add(60 * 60 * 24 * 365 * 1000 * 1000 * 1000 * time.Duration(2))
	//
	//pemStr := cryptolicensing.ReadCertPemFromFile("./rootSSCert.pem")
	//
	//rootCert, err := cryptolicensing.LoadCertFromPEM([]byte(pemStr))
	//if err != nil {
	//	panic(err)
	//}
	//
	//priv, err := cryptolicensing.GenerateKeyPair()
	//if err != nil {
	//	log.Fatalf("error generating pub/priv key pair")
	//}
	//
	//err = cryptolicensing.SavePrivateKeyAsPEM(priv, "./intermediatePKey.pem")
	//if err != nil {
	//	log.Fatalf("error saving private key to file: %s", err)
	//}
	//
	//derBytes, err := cryptolicensing.Generatex509CertChain(now, then, rootCert, priv)
	//
	//if err != nil {
	//	panic(err)
	//}
	//
	//err = cryptolicensing.SaveCertAsPEM(derBytes, "./intermediateCert.pem")
	//if err != nil {
	//	log.Fatalf("error saving cert to file: %s", err)
	//}
	//

	// First, create the set of root certificates. For this example we only
	// have one. It's also possible to omit this in order to use the
	// default root set of the current operating system.
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM([]byte(rootPEM))
	if !ok {
		panic("failed to parse root certificate")
	}

	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		panic("failed to parse certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic("failed to parse certificate: " + err.Error())
	}

	opts := x509.VerifyOptions{
		Roots: roots,
	}

	if _, err := cert.Verify(opts); err != nil {
		panic("failed to verify certificate: " + err.Error())
	}

	fmt.Println("SUCCESS!!!!!!!!")

}

func CreateCertKeyPair(certPath, pKeyPath string, certYears int) {
	// Generate Pub/Priv key pair.
	priv, err := cryptolicensing.GenerateKeyPair()
	if err != nil {
		log.Fatalf("error generating pub/priv key pair")
	}

	err = cryptolicensing.SavePrivateKeyAsPEM(priv, pKeyPath)
	if err != nil {
		log.Fatalf("error saving private key to file: %s", err)
	}

	// Generate x.509 certificate.
	now := time.Now()
	// Valid for one year from now.
	then := now.Add(60 * 60 * 24 * 365 * 1000 * 1000 * 1000 * time.Duration(certYears))
	derBytes, err := cryptolicensing.Generatex509Cert(now, then, priv)
	if err != nil {
		log.Fatalf("error generating x.509 certificate: %s", err)
	}

	//err = cryptolicensing.SaveCertToFile(derBytes, "test_evil_cert.cer")
	//if err != nil {
	//	log.Fatalf("error saving cert to file: %s", err)
	//}

	err = cryptolicensing.SaveCertAsPEM(derBytes, certPath)
	if err != nil {
		log.Fatalf("error saving cert to file: %s", err)
	}
}
