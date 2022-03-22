// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package resource

import (
	"crypto/sha1"
	"fmt"
)

const (
	ElasticsearchConfigMapName                        = "tigera-secure-elasticsearch"
	ElasticsearchCertSecret                           = "tigera-secure-es-http-certs-public"
	KibanaCertSecret                                  = "tigera-secure-kb-http-certs-public"
	ESGatewayCertSecret                               = "tigera-secure-es-gateway-http-certs-public"
	OperatorNamespace                                 = "tigera-operator"
	TigeraElasticsearchNamespace                      = "tigera-elasticsearch"
	DefaultTSEEInstanceName                           = "tigera-secure"
	OIDCUsersConfigMapName                            = "tigera-known-oidc-users"
	OIDCUsersEsSecreteName                            = "tigera-oidc-users-elasticsearch-credentials"
	LicenseName                                       = "default"
	CalicoNamespaceName                               = "calico-system"
	ActiveOperatorConfigMapName                       = "active-operator"
	ImageAssuranceConfigMapName                       = "tigera-image-assurance-config"
	ImageAssuranceAPICertPairSecretName               = "tigera-image-assurance-api-cert-pair"
	ImageAssuranceAPICertSecretName                   = "tigera-image-assurance-api-cert"
	ImageAssuranceNameSpaceName                       = "tigera-image-assurance"
	AdmissionControllerResourceName                   = "admission-controller-api-access"
	ImageAssuranceAdmissionControllerRoleName         = "tigera-image-assurance-admission-controller-role"
	ManagedIAAdmissionControllerResourceName          = "tigera-image-assurance-admission-controller-api-access"
	ManagementIAAdmissionControllerResourceNameFormat = "tigera-image-assurance-%s-admission-controller-api-access"
)

func CreateHashFromObject(obj interface{}) (string, error) {
	h := sha1.New()
	_, err := h.Write([]byte(fmt.Sprintf("%q", obj)))
	return fmt.Sprintf("%x", h.Sum(nil)), err
}
