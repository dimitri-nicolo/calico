// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package resource

import (
	"crypto/sha1"
	"fmt"
)

const (
	ElasticsearchConfigMapName   = "tigera-secure-elasticsearch"
	ElasticsearchCertSecret      = "tigera-secure-es-http-certs-public"
	ESGatewayCertSecret          = "tigera-secure-es-gateway-http-certs-public"
	OperatorNamespace            = "tigera-operator"
	TigeraElasticsearchNamespace = "tigera-elasticsearch"
	DefaultTSEEInstanceName      = "tigera-secure"
	OIDCUsersConfigMapName       = "tigera-known-oidc-users"
	OIDCUsersEsSecreteName       = "tigera-oidc-users-elasticsearch-credentials"
	LicenseName                  = "default"
)

func CreateHashFromObject(obj interface{}) (string, error) {
	h := sha1.New()
	_, err := h.Write([]byte(fmt.Sprintf("%q", obj)))
	return fmt.Sprintf("%x", h.Sum(nil)), err
}
