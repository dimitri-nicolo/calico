package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/coreos/go-oidc"
	log "github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2"
	"k8s.io/apiserver/pkg/authentication/user"

	"github.com/projectcalico/apiserver/pkg/authentication"
)

const (
	signingAlg           = "RS256"
	defaultUsernameClaim = "sub"
	defaultGroupsClaim   = "groups"
	noUsernamePrefix     = "-"
)

type dexAuthenticator struct {
	// The issuer as it is added to the JWTs. Ex. https://tigera-manager/dex
	issuer string

	clientID string

	usernameClaim string

	groupsClaim string

	usernamePrefix *string

	groupsPrefix string

	verifier *oidc.IDTokenVerifier
}

// DexOption can be provided to NewDexAuthenticator to configure the authenticator.
type DexOption func(*dexAuthenticator) error

// NewDexAuthenticator creates an authenticator that uses DexIdp to validate authorization headers.
func NewDexAuthenticator(issuer, clientID, usernameClaim string, options ...DexOption) (authentication.Authenticator, error) {
	if issuer == "" {
		return nil, errors.New("issuer is a required field")
	}

	if clientID == "" {
		return nil, errors.New("clientID is a required field")
	}

	dex := &dexAuthenticator{
		issuer:        issuer,
		clientID:      clientID,
		groupsClaim:   defaultGroupsClaim,
		usernameClaim: usernameClaim,
	}

	if usernameClaim == "" {
		dex.usernameClaim = defaultUsernameClaim
	}

	for _, option := range options {
		if err := option(dex); err != nil {
			return nil, err
		}
	}

	if dex.usernamePrefix == nil {
		if err := WithUsernamePrefix("")(dex); err != nil {
			return nil, err
		}
	}

	return dex, nil
}

// WithJWKSURL The authenticator will validate JWT signatures based on the public keys that are available at this URL.
// Cannot be used in combination with WithKeySet().
func WithJWKSURL(jwksURL string) DexOption {
	return func(d *dexAuthenticator) error {
		if d.verifier != nil {
			return errors.New("can only use one of: [WithKeySet(), WithJWKSURL()]")
		}

		d.verifier = oidc.NewVerifier(d.issuer,
			oidc.NewRemoteKeySet(context.Background(), jwksURL),
			&oidc.Config{
				ClientID:             d.clientID,
				SkipClientIDCheck:    false,
				SkipExpiryCheck:      false,
				SupportedSigningAlgs: []string{signingAlg},
				SkipIssuerCheck:      false,
			})
		return nil
	}
}

// WithKeySet Provide your own keyset to validate JWT signatures. Useful for testing. Cannot be used in combination with
// WithJWKSURL().
func WithKeySet(keySet oidc.KeySet) DexOption {
	return func(d *dexAuthenticator) error {
		if d.verifier != nil {
			return errors.New("can only use one of the following options: [WithKeySet(), WithJWKSURL()]")
		}

		d.verifier = oidc.NewVerifier(d.issuer,
			keySet,
			&oidc.Config{
				ClientID:             d.clientID,
				SkipClientIDCheck:    false,
				SkipExpiryCheck:      false,
				SupportedSigningAlgs: []string{signingAlg},
				SkipIssuerCheck:      false,
			})
		return nil
	}
}

// WithGroupsClaim set the claim to extract groups from a JWT. Default: 'groups'.
func WithGroupsClaim(groupsClaim string) DexOption {
	return func(d *dexAuthenticator) error {
		d.groupsClaim = groupsClaim
		return nil
	}
}

// The value passed in as a prefix will be modified according to the kubernetes specs for UsernamePrefix for backwards
// compatibility purposes.
// See: kubernetes/pkg/kubeapiserver/authenticator/config.go or
// https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/ for more details.
func WithUsernamePrefix(usernamePrefix string) DexOption {
	return func(d *dexAuthenticator) error {
		prefix := usernamePrefix

		if usernamePrefix == "" && d.usernameClaim != "email" {
			// Old behavior. If a usernamePrefix isn't provided, prefix all claims other than "email"
			// with the issuerURL.
			//
			// See https://github.com/kubernetes/kubernetes/issues/31380
			prefix = d.issuer + "#"
		}

		if usernamePrefix == noUsernamePrefix {
			// Special value indicating usernames shouldn't be prefixed.
			prefix = ""
		}

		d.usernamePrefix = &prefix
		return nil
	}
}

// WithGroupsPrefix adds a prefix to every extracted group from a JWT.
func WithGroupsPrefix(groupsPrefix string) DexOption {
	return func(d *dexAuthenticator) error {
		d.groupsPrefix = groupsPrefix
		return nil
	}
}

// Authenticate returns user info if the authHeader has a valid token issued by Dex.
// Returns HTTP code 421 if the issuer is not Dex.
// Returns an error if the auth header does not contain a valid credential.
func (d *dexAuthenticator) Authenticate(authHeader string) (user.Info, int, error) {

	// First we determine if the auth header is associated with this authenticator. Otherwise, return a 421.
	if len(authHeader) == 0 {
		return nil, 401, errors.New("authorization header is missing")
	}

	jwt := strings.TrimPrefix(authHeader, "Bearer ")
	if len(jwt) == len(authHeader) {
		return nil, 421, errors.New("not a dex header: header should start with 'Bearer '")
	}
	jwt = strings.TrimSpace(jwt)

	if strings.Count(authHeader, ".") != 2 {
		return nil, 421, errors.New("not a dex header: token should should adhere to the following format (JWT): <b64-hdr>.<b64-payload>.<b64-sig>")
	}

	payload := strings.Split(authHeader, ".")[1]
	var tokenPayloadMap map[string]interface{}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, 421, errors.New("not a dex header: JWT payload cannot be base64 decoded")
	}

	err = json.Unmarshal(payloadBytes, &tokenPayloadMap)
	if err != nil {
		return nil, 421, errors.New("not a dex header: JWT payload does not contain base64 encoded json")
	}

	iss := tokenPayloadMap["iss"].(string)
	if iss != d.issuer {
		return nil, 421, errors.New("not a dex header: issuer of JWT does not match the issuer url of dex")
	}

	// Now that we know the token was issued by dex, we can verify if it is (still) valid and extract the user.
	_, err = jose.ParseSigned(jwt)
	if err != nil {
		return nil, 401, errors.New("dex token has an invalid signature")
	}

	idTkn, err := d.verifier.Verify(context.Background(), jwt)
	if err != nil {
		return nil, 401, err
	}

	var claims map[string]interface{}
	if err := idTkn.Claims(&claims); err != nil {
		return nil, 500, err
	}

	usr, ok := claims[d.usernameClaim]
	if !ok {
		return nil, 401, fmt.Errorf("unable to extract username from JWT using claim %s", d.usernameClaim)
	}

	username, ok := usr.(string)
	if !ok {
		return nil, 400, errors.New("the username should be of type string")
	}

	if username == "" {
		return nil, 401, errors.New("no user found in JWT")
	}
	username = fmt.Sprintf("%s%s", *d.usernamePrefix, usr)
	groups := []string{}

	if claims[d.groupsClaim] != nil {
		groupsClaims, ok := claims[d.groupsClaim].([]interface{})
		if !ok {
			return nil, 400, errors.New("unexpected type for groups claim")
		}

		for _, group := range groupsClaims {
			groupStr, ok := group.(string)
			if !ok {
				return nil, 400, errors.New("unexpected type for element in groups claim")
			}
			groups = append(groups, fmt.Sprintf("%s%s", d.groupsPrefix, groupStr))
		}
	}

	// Setting issuer and subject as Extra, this can be used to identify userInfo authenticated by dex
	extra := make(map[string][]string)
	extra["iss"] = []string{iss}
	if subClaim, ok := claims[defaultUsernameClaim].(string); !ok {
		log.Warn("subject claim is not of type string")
	} else {
		extra["sub"] = []string{subClaim}
	}

	return &user.DefaultInfo{
		Name:   username,
		Groups: groups,
		Extra:  extra,
	}, 200, nil
}
