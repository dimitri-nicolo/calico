// Copyright (c) 2022 Tigera, Inc. All rights reserved.
package auth

import (
	log "github.com/sirupsen/logrus"

	lmaauth "github.com/projectcalico/calico/lma/pkg/auth"
	"github.com/projectcalico/calico/ts-queryserver/queryserver/config"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// GetJWTAuth returns an lma JWT authenticator.
func GetJWTAuth(cfg *config.Config, restcfg *rest.Config, k8sclient kubernetes.Interface) (lmaauth.JWTAuth, error) {
	var options []lmaauth.JWTAuthOption
	if cfg.DexEnabled {
		log.Info("Configuring Dex for authentication")
		opts := []lmaauth.DexOption{
			lmaauth.WithGroupsClaim(cfg.OIDCAuthGroupsClaim),
			lmaauth.WithJWKSURL(cfg.OIDCAuthJWKSURL),
			lmaauth.WithUsernamePrefix(cfg.OIDCAuthUsernamePrefix),
			lmaauth.WithGroupsPrefix(cfg.OIDCAuthGroupsPrefix),
		}
		dex, err := lmaauth.NewDexAuthenticator(
			cfg.OIDCAuthIssuer,
			cfg.OIDCAuthClientID,
			cfg.OIDCAuthUsernameClaim,
			opts...)
		if err != nil {
			log.WithError(err).Fatal("Unable to add an issuer to the authenticator")
		}
		options = append(options, lmaauth.WithAuthenticator(cfg.OIDCAuthIssuer, dex))
	}

	// Configure authenticator.
	return lmaauth.NewJWTAuth(restcfg, k8sclient, options...)
}
