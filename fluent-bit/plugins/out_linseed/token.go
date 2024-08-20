// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package main

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	tokenExpiration = 24 * time.Hour
	tokenRenewal    = 15 * time.Minute
)

func GetToken(cfg *Config) (string, error) {
	if time.Until(cfg.expiration) < tokenRenewal {
		logrus.Infof("token expired for serviceaccount %q", cfg.serviceAccountName)
		token, expiration, err := getServiceAccountToken(cfg.clientset.CoreV1(), corev1.NamespaceDefault, cfg.serviceAccountName)
		if err != nil {
			return "", err
		}
		cfg.expiration = expiration
		cfg.token = token
		logrus.Infof("successfully renewed token for serviceaccount %q", cfg.serviceAccountName)
	}

	return cfg.token, nil
}

func getServiceAccountToken(coreV1Client v1.CoreV1Interface, namespace, serviceAccountName string) (string, time.Time, error) {
	seconds := int64(tokenExpiration.Seconds())
	tokenRequest := &authv1.TokenRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: namespace,
		},
		Spec: authv1.TokenRequestSpec{
			Audiences:         []string{},
			ExpirationSeconds: &seconds,
		},
	}

	tokenResponse, err := coreV1Client.ServiceAccounts(namespace).CreateToken(context.Background(), serviceAccountName, tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", time.Time{}, err
	}

	token := tokenResponse.Status.Token
	expiration := tokenResponse.Status.ExpirationTimestamp.Time

	return token, expiration, nil
}
