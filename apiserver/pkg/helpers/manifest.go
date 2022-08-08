// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package helpers

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"text/template"
)

// installationManifestTemplate represents the template used to generate a manifest that will be applied on managed
// cluster. Applying the manifest will result in the creation of the ManagementClusterConnection CR and the Secret
// tigera-managed-cluster-connection. The secret must be created in the active
// operator's namespace.
//The template contains the following customizable entries:
// - {{.CACert}}
// - {{.Cert}}
// - {{.PrivateKey}}
// - {{.OperatorNamespace}}
//By design managementClusterAddr is intended to be left unfilled (until the user downloads this manifest and fills it).
//In the future, we will autofill this field using user-facing config
const installationManifestTemplate = `
# Once applied to your managed cluster, a deployment is created to establish a secure tcp connection
# with the management cluster.

apiVersion: operator.tigera.io/v1
kind: ManagementClusterConnection
metadata:
  name: tigera-secure
spec:
  # ManagementClusterAddr should be the externally reachable address to which your managed cluster
  # will connect. Valid examples are: "0.0.0.0:31000", "example.com:32000", "[::1]:32500"
  managementClusterAddr: "{{.ManagementClusterAddr}}"

  tls:
    ca: "{{.ManagementClusterCAType}}"
---

apiVersion: v1
kind: Secret
metadata:
  name: tigera-managed-cluster-connection
  namespace: {{.OperatorNamespace}}
type: Opaque
data:
  # This is the certificate of the management cluster side of the tunnel.
  management-cluster.crt: {{.CACert | base64}}
  # The certificate and private key that are created and signed by the CA in the management cluster.
  managed-cluster.crt: {{.Cert | base64}}
  managed-cluster.key: {{.PrivateKey | base64}}
`

// manifestConfig renders manifest based on a predefined template
type manifestConfig struct {
	CACert                  string
	Cert                    string
	PrivateKey              string
	ManagementClusterAddr   string
	ManagementClusterCAType string
	OperatorNamespace       string
}

// keyPEMEncode encodes a crypto.Signer as a PEM block
func keyPEMEncode(key crypto.Signer) []byte {
	var block = &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key.(*rsa.PrivateKey)),
	}

	return pem.EncodeToMemory(block)
}

// certPEMEncode encodes a x509.Certificate as a PEM block
func certPEMEncode(cert *x509.Certificate) []byte {
	var block = &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}

	return pem.EncodeToMemory(block)
}

// InstallationManifest generates an installation manifests that will populate the field
// for a managed cluster upon creation
func InstallationManifest(certCA, cert *x509.Certificate, key crypto.Signer, managementClusterAddr, managementClusterCAType, operatorNamespace string) string {
	var manifest bytes.Buffer
	var tmpl *template.Template

	funcMap := template.FuncMap{
		"base64": func(text string) string {
			return base64.StdEncoding.EncodeToString([]byte(text))
		},
	}
	var text = installationManifestTemplate

	tmpl, _ = template.New("ClusterManifest").Funcs(funcMap).Parse(text)

	manifestConfig := &manifestConfig{
		CACert:                  string(certPEMEncode(certCA)),
		Cert:                    string(certPEMEncode(cert)),
		PrivateKey:              string(keyPEMEncode(key)),
		ManagementClusterAddr:   managementClusterAddr,
		ManagementClusterCAType: managementClusterCAType,
		OperatorNamespace:       operatorNamespace,
	}
	_ = tmpl.Execute(&manifest, manifestConfig)
	return manifest.String()
}
