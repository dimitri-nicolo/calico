// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package elasticsearch

import (
	"context"
	"crypto/x509"
	"fmt"

	"github.com/projectcalico/kube-controllers/pkg/resource"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ClientCredentialsFromK8sCLI uses the given kubernetes.Clientset to retrieve the username, password, and root certificates
// needed to authenticate with the elasticsearch cluster.
func ClientCredentialsFromK8sCLI(k8sCLI kubernetes.Interface) (string, string, *x509.CertPool, error) {
	ctx := context.Background()

	esSecret, err := k8sCLI.CoreV1().Secrets(resource.TigeraElasticsearchNamespace).Get(ctx, resource.ElasticsearchUserSecret, metav1.GetOptions{})
	if err != nil {
		return "", "", nil, err
	}
	esPublicCert, err := k8sCLI.CoreV1().Secrets(resource.OperatorNamespace).Get(ctx, resource.ElasticsearchCertSecret, metav1.GetOptions{})
	if err != nil {
		return "", "", nil, err
	}

	roots, err := getESRoots(esPublicCert)

	return "elastic", string(esSecret.Data["elastic"]), roots, err
}

func getESRoots(esCertSecret *corev1.Secret) (*x509.CertPool, error) {
	rootPEM, exists := esCertSecret.Data["tls.crt"]
	if !exists {
		return nil, fmt.Errorf("couldn't find tls.crt in Elasticsearch secret")
	}

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(rootPEM)
	if !ok {
		return nil, fmt.Errorf("failed to parse root certificate")
	}

	return roots, nil
}
