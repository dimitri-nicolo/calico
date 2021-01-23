// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package resource

import (
	"crypto/sha1"
	"fmt"
)

const (
	ElasticsearchUserSecret      = "tigera-secure-es-elastic-user"
	ElasticsearchConfigMapName   = "tigera-secure-elasticsearch"
	ElasticsearchCertSecret      = "tigera-secure-es-http-certs-public"
	KibanaCertSecret             = "tigera-secure-kb-http-certs-public"
	OperatorNamespace            = "tigera-operator"
	TigeraElasticsearchNamespace = "tigera-elasticsearch"
	DefaultTSEEInstanceName      = "tigera-secure"
	ElasticsearchServiceURL      = "https://tigera-secure-es-http.tigera-elasticsearch.svc:9200"
	OIDCUsersConfigMapName       = "tigera-known-oidc-users"
	OIDCUsersEsSecreteName       = "tigera-oidc-users-elasticsearch-credentials"
)

func CreateHashFromObject(obj interface{}) (string, error) {
	h := sha1.New()
	_, err := h.Write([]byte(fmt.Sprintf("%q", obj)))
	return fmt.Sprintf("%x", h.Sum(nil)), err
}
