// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package server

import (
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"io"
	"text/template"

	"github.com/pkg/errors"
	"github.com/tigera/voltron/internal/pkg/utils"
)

type manifestConfig struct {
	VoltronURL      string
	VoltronCert     string
	GuardianTLSKey  string
	GuardianTLSCert string
}

// renderer renders manifest based on a predefined template
type renderer struct {
	template      *template.Template
	serverCert    string
	serverAddress string
}

type manifestRenderer func(wr io.Writer, cert *x509.Certificate, key crypto.Signer) error

// newRenderer creates a new Renderer for manifests
func newRenderer(content string, serverAddress string,
	serverCert *x509.Certificate) (manifestRenderer, error) {
	funcMap := template.FuncMap{
		"base64": encodeBase64,
	}

	t, err := template.New("ClusterManifest").Funcs(funcMap).Parse(content)
	if err != nil {
		return nil, errors.New("Could not parse template content")
	}

	pemCert := string(utils.CertPEMEncode(serverCert))
	r := &renderer{
		template:      t,
		serverCert:    pemCert,
		serverAddress: serverAddress,
	}

	return r.RenderManifest, nil
}

func encodeBase64(text string) string {
	return base64.StdEncoding.EncodeToString([]byte(text))
}

// RenderManifest writes the manifest by filling out the macros from the template
func (r *renderer) RenderManifest(wr io.Writer, cert *x509.Certificate, key crypto.Signer) error {
	keyStr, err := utils.KeyPEMEncode(key)

	if err != nil {
		return errors.WithMessage(err, "could not extract key to apply to template")
	}

	m := manifestConfig{
		VoltronURL:      r.serverAddress,
		VoltronCert:     r.serverCert,
		GuardianTLSCert: string(utils.CertPEMEncode(cert)),
		GuardianTLSKey:  string(keyStr),
	}

	err = r.template.Execute(wr, m)
	if err != nil {
		return errors.Errorf("could not apply template: %s", err)
	}

	return nil
}
