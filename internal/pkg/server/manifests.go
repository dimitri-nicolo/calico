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

	log "github.com/sirupsen/logrus"
)

type manifestConfig struct {
	VoltronURL      string
	VoltronCert     string
	GuardianTLSKey  string
	GuardianTLSCert string
}

// Renderer renders manifest based on a predefined template
type Renderer struct {
	template      *template.Template
	serverCert    string
	serverAddress string
}

// NewRenderer creates a new Renderer for manifests
func NewRenderer(content string, serverAddress string, serverCert *x509.Certificate) (*Renderer, error) {
	funcMap := template.FuncMap{
		"base64": encodeBase64,
	}

	t, err := template.New("ClusterManifest").Funcs(funcMap).Parse(content)
	if err != nil {
		return nil, errors.New("Could not parse template content")
	}

	pemCert := string(utils.CertPEMEncode(serverCert))
	return &Renderer{template: t, serverCert: pemCert, serverAddress: serverAddress}, nil
}

func encodeBase64(text string) string {
	return base64.StdEncoding.EncodeToString([]byte(text))
}

// RenderManifest writes the manifest by filling out the macros from the template
func (r *Renderer) RenderManifest(wr io.Writer, cert *x509.Certificate, key crypto.Signer) bool {
	keyStr, err := utils.KeyPEMEncode(key)

	if err != nil {
		log.Warnf("Could not extract key to apply to template %v", err)
		return false
	}

	m := manifestConfig{
		VoltronURL:      r.serverAddress,
		VoltronCert:     r.serverCert,
		GuardianTLSCert: string(utils.CertPEMEncode(cert)),
		GuardianTLSKey:  string(keyStr),
	}

	err = r.template.Execute(wr, m)
	if err != nil {
		log.Warnf("Could not apply template %v", err)
		return false
	}

	return true
}
