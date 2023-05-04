// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package testutils

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"github.com/projectcalico/calico/linseed/pkg/controller/token"
)

func genToken(subject, issuer, issuerName string, expire time.Duration, key *rsa.PrivateKey) ([]byte, error) {
	expirationTime := time.Now().Add(expire)

	claims := &jwt.RegisteredClaims{
		Subject:   subject,
		Issuer:    issuer,
		Audience:  jwt.ClaimStrings{issuerName},
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(expirationTime),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(key)
	if err != nil {
		return nil, err
	}
	return []byte(tokenString), err
}

func LinseedToken(tenant, cluster, namespace, name string, expire time.Duration, key *rsa.PrivateKey) ([]byte, error) {
	subj := fmt.Sprintf("%s:%s:%s:%s", tenant, cluster, namespace, name)
	issuerName := fmt.Sprintf("tigera-linseed-%s-token", name)
	return genToken(subj, token.LinseedIssuer, issuerName, expire, key)
}

func K8sToken(namespace, name string, expire time.Duration, key *rsa.PrivateKey) ([]byte, error) {
	issuer := "https://kubernetes.default.svc.cluster.local"
	issuerName := "https://kubernetes.default.svc.cluster.local"
	subject := fmt.Sprintf("system:serviceaccount:%s:%s", namespace, name)
	return genToken(subject, issuer, issuerName, expire, key)
}
