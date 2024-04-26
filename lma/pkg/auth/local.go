// Copyright (c) 2023 Tigera, Inc. All rights reserved.
package auth

import (
	"crypto"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/SermoDigital/jose/jws"
	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v4"
	"k8s.io/apiserver/pkg/authentication/user"
)

// NewLocalAuthenticator returns an Authenticator that can authenticate tokens locally using
// the given public key.
func NewLocalAuthenticator(iss string, key crypto.PublicKey, cp ClaimParser) Authenticator {
	return &localAuthenticator{
		issuer:      iss,
		key:         key,
		claimParser: cp,
	}
}

type ClaimParser func(jwt.Claims) (*user.DefaultInfo, error)

type localAuthenticator struct {
	issuer      string
	key         crypto.PublicKey
	claimParser ClaimParser
}

func (a *localAuthenticator) Authenticate(r *http.Request) (user.Info, int, error) {
	reqJWT, err := jws.ParseJWTFromRequest(r)
	if err != nil {
		return nil, 401, jws.ErrNoTokenInRequest
	}
	tokenPayloadMap := reqJWT.Claims()

	iss := tokenPayloadMap["iss"].(string)
	if iss != a.issuer {
		return nil, 421, fmt.Errorf("token was not issued by %s", a.issuer)
	}

	// Strip the "Bearer " part of the token.
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, 403, fmt.Errorf("no bearer token provided")
	}
	tokenString := authHeader[7:]
	tokenString = strings.TrimSpace(tokenString)

	// Now that we know the token was issued by us, we can check if it is (still) valid and extract the user.
	_, err = jose.ParseSigned(tokenString, []jose.SignatureAlgorithm{jose.RS256, jose.RS384, jose.RS512, jose.ES256, jose.ES384, jose.ES512})
	if err != nil {
		return nil, 401, fmt.Errorf("token has an invalid signature")
	}

	parsedToken, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return a.key, nil
	})
	if err != nil {
		return nil, 403, err
	}

	if err = parsedToken.Claims.Valid(); err != nil {
		return nil, 403, err
	} else if !parsedToken.Claims.(*jwt.RegisteredClaims).VerifyExpiresAt(time.Now(), true) {
		// We require a time claim is included, so check this explicitly.
		return nil, 403, fmt.Errorf("token is expired")
	}

	if a.claimParser == nil {
		return nil, 500, fmt.Errorf("no claim parser provided")
	}
	userInfo, err := a.claimParser(parsedToken.Claims)
	if err != nil {
		return nil, 500, err
	}

	return userInfo, 200, nil
}
