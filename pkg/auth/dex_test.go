package auth_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/projectcalico/apiserver/pkg/authentication"
	"github.com/stretchr/testify/mock"
	"github.com/tigera/lma/pkg/auth"
)

var _ = Describe("Test dex authenticator and options", func() {
	const (
		iss            = "https://127.0.0.1:9443/dex"
		exp            = 9600964803 //Very far in the future
		email          = "rene@tigera.io"
		usernamePrefix = "my-user:"
		usernameClaim  = "email"
		clientID       = "tigera-manager"
		prefixedGroup  = "my-group:admins"
		prefixedUser   = "my-user:rene@tigera.io"

		badIss      = "https:/accounts.google.com"
		badExp      = 1600964803 //Recently expired
		badClientID = "starbucks"
	)

	var dex authentication.Authenticator
	var err error
	var keySet *testKeySet

	BeforeEach(func() {
		keySet = &testKeySet{}
		opts := []auth.DexOption{
			auth.WithGroupsClaim("groups"),
			auth.WithUsernamePrefix(usernamePrefix),
			auth.WithGroupsPrefix("my-group:"),
			auth.WithKeySet(keySet),
		}
		dex, err = auth.NewDexAuthenticator(iss, clientID, usernameClaim, opts...)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should authenticate a valid dex user", func() {
		hdr, payload := authHeader(iss, email, clientID, exp)
		keySet.On("VerifySignature", mock.Anything, strings.TrimSpace(strings.TrimPrefix(hdr, "Bearer "))).Return(payload, nil)
		usr, stat, err := dex.Authenticate(hdr)
		Expect(err).NotTo(HaveOccurred())
		Expect(usr).NotTo(BeNil())
		Expect(usr.GetName()).To(Equal(prefixedUser))
		Expect(usr.GetGroups()[0]).To(Equal(prefixedGroup))
		Expect(usr.GetExtra()["iss"]).To(Equal([]string{iss}))
		Expect(usr.GetExtra()["sub"]).To(Equal([]string{"ChUxMDkxMzE"}))
		Expect(stat).To(Equal(200))
	})

	It("should reject an invalid issuer", func() {
		hdr, payload := authHeader(badIss, email, clientID, exp)
		keySet.On("VerifySignature", mock.Anything, strings.TrimSpace(strings.TrimPrefix(hdr, "Bearer "))).Return(payload, nil)
		usr, stat, err := dex.Authenticate(hdr)
		Expect(err).NotTo(BeNil())
		Expect(usr).To(BeNil())
		Expect(stat).To(Equal(421))
	})

	It("should reject an invalid clientID", func() {
		hdr, payload := authHeader(iss, email, badClientID, exp)
		keySet.On("VerifySignature", mock.Anything, strings.TrimSpace(strings.TrimPrefix(hdr, "Bearer "))).Return(payload, nil)
		usr, stat, err := dex.Authenticate(hdr)
		Expect(err).NotTo(BeNil())
		Expect(usr).To(BeNil())
		Expect(stat).To(Equal(401))
	})

	It("should reject an expired token", func() {
		hdr, payload := authHeader(iss, email, clientID, badExp)
		keySet.On("VerifySignature", mock.Anything, strings.TrimSpace(strings.TrimPrefix(hdr, "Bearer "))).Return(payload, nil)
		usr, stat, err := dex.Authenticate(hdr)
		Expect(err).NotTo(BeNil())
		Expect(usr).To(BeNil())
		Expect(stat).To(Equal(401))
	})

	It("should reject an invalid signature", func() {
		hdr, _ := authHeader(iss, email, clientID, exp)
		keySet.On("VerifySignature", mock.Anything, strings.TrimSpace(strings.TrimPrefix(hdr, "Bearer "))).Return(nil, errors.New("sig error"))
		usr, stat, err := dex.Authenticate(hdr)
		Expect(err).NotTo(BeNil())
		Expect(usr).To(BeNil())
		Expect(stat).To(Equal(401))
	})
})

var _ = Describe("Test dex username prefixes", func() {
	const (
		iss      = "https://127.0.0.1:9443/dex"
		exp      = 9600964803 //Very far in the future
		email    = "rene@tigera.io"
		clientID = "tigera-manager"
	)

	hdr, payload := authHeader(iss, email, clientID, exp)
	var opts []auth.DexOption

	BeforeEach(func() {

		keySet := &testKeySet{}
		keySet.On("VerifySignature", mock.Anything, strings.TrimSpace(strings.TrimPrefix(hdr, "Bearer "))).Return(payload, nil)
		opts = []auth.DexOption{
			auth.WithGroupsClaim("groups"),
			auth.WithGroupsPrefix("my-groups"),
			auth.WithKeySet(keySet),
		}
	})

	It("should prepend the prefix to the username", func() {
		prefix := "Howdy, "
		opts = append(opts, auth.WithUsernamePrefix(prefix))
		dx, err := auth.NewDexAuthenticator(iss, clientID, "name", opts...)
		Expect(err).NotTo(HaveOccurred())
		usr, stat, err := dx.Authenticate(hdr)

		Expect(err).NotTo(HaveOccurred())
		Expect(usr).NotTo(BeNil())
		Expect(usr.GetName()).To(Equal("Howdy, Rene Dekker"))
		Expect(stat).To(Equal(200))
	})

	It("should prepend nothing to the username", func() {
		prefix := "-"
		opts = append(opts, auth.WithUsernamePrefix(prefix))
		dx, err := auth.NewDexAuthenticator(iss, clientID, "name", opts...)
		Expect(err).NotTo(HaveOccurred())
		usr, stat, err := dx.Authenticate(hdr)

		Expect(err).NotTo(HaveOccurred())
		Expect(usr).NotTo(BeNil())
		Expect(usr.GetName()).To(Equal("Rene Dekker"))
		Expect(stat).To(Equal(200))
	})

	It("should prepend issuer to the username", func() {
		prefix := ""
		opts = append(opts, auth.WithUsernamePrefix(prefix))
		dx, err := auth.NewDexAuthenticator(iss, clientID, "name", opts...)
		Expect(err).NotTo(HaveOccurred())
		usr, stat, err := dx.Authenticate(hdr)

		Expect(err).NotTo(HaveOccurred())
		Expect(usr).NotTo(BeNil())
		Expect(usr.GetName()).To(Equal(fmt.Sprintf("%s#Rene Dekker", iss)))
		Expect(stat).To(Equal(200))
	})

	It("should prepend the prefix to the username (email claim)", func() {
		prefix := "Howdy, "
		opts = append(opts, auth.WithUsernamePrefix(prefix))
		dx, err := auth.NewDexAuthenticator(iss, clientID, "email", opts...)
		Expect(err).NotTo(HaveOccurred())
		usr, stat, err := dx.Authenticate(hdr)

		Expect(err).NotTo(HaveOccurred())
		Expect(usr).NotTo(BeNil())
		Expect(usr.GetName()).To(Equal("Howdy, rene@tigera.io"))
		Expect(stat).To(Equal(200))
	})

	It("should prepend nothing to the username (email claim)(1/2)", func() {
		prefix := "-"
		opts = append(opts, auth.WithUsernamePrefix(prefix))
		dx, err := auth.NewDexAuthenticator(iss, clientID, "email", opts...)
		Expect(err).NotTo(HaveOccurred())
		usr, stat, err := dx.Authenticate(hdr)

		Expect(err).NotTo(HaveOccurred())
		Expect(usr).NotTo(BeNil())
		Expect(usr.GetName()).To(Equal("rene@tigera.io"))
		Expect(stat).To(Equal(200))
	})

	It("should prepend nothing to the username (email claim)(2/2)", func() {
		prefix := ""
		opts = append(opts, auth.WithUsernamePrefix(prefix))
		dx, err := auth.NewDexAuthenticator(iss, clientID, "email", opts...)
		Expect(err).NotTo(HaveOccurred())
		usr, stat, err := dx.Authenticate(hdr)

		Expect(err).NotTo(HaveOccurred())
		Expect(usr).NotTo(BeNil())
		Expect(usr.GetName()).To(Equal("rene@tigera.io"))
		Expect(stat).To(Equal(200))
	})

	It("should prepend the right prefix to the username if no prefix option was specified", func() {
		dx, err := auth.NewDexAuthenticator(iss, clientID, "name", opts...)
		Expect(err).NotTo(HaveOccurred())
		usr, stat, err := dx.Authenticate(hdr)

		Expect(err).NotTo(HaveOccurred())
		Expect(usr).NotTo(BeNil())
		Expect(usr.GetName()).To(Equal(fmt.Sprintf("%s#Rene Dekker", iss)))
		Expect(stat).To(Equal(200))
	})

	It("should prepend the right prefix to the username if no prefix option was specified (email claim)", func() {
		dx, err := auth.NewDexAuthenticator(iss, clientID, "email", opts...)
		Expect(err).NotTo(HaveOccurred())
		usr, stat, err := dx.Authenticate(hdr)

		Expect(err).NotTo(HaveOccurred())
		Expect(usr).NotTo(BeNil())
		Expect(usr.GetName()).To(Equal("rene@tigera.io"))
		Expect(stat).To(Equal(200))
	})

})

type testKeySet struct {
	mock.Mock
}

// Test Verify method.
func (t *testKeySet) VerifySignature(ctx context.Context, jwt string) ([]byte, error) {
	args := t.Called(ctx, jwt)
	err := args.Get(1)
	if err != nil {
		return nil, err.(error)
	}
	return args.Get(0).([]byte), nil
}

func authHeader(issuer, email, clientID string, exp int) (string, []byte) {
	hdrhdr := "eyJhbGciOiJSUzI1NiIsImtpZCI6Ijk3ODM2YzRiMjdmN2M3ZmVjMjk1MTk0NTFkNDc5MmUyNjQ4M2RmYWUifQ" // rs256 header
	payload := map[string]interface{}{
		"iss":            issuer,
		"sub":            "ChUxMDkxMzE",
		"aud":            clientID,
		"exp":            exp, //Very far in the future
		"iat":            1600878403,
		"nonce":          "35e32c66028243f592cc3103c7c2dfb2",
		"at_hash":        "jOq0F62t_NE9a3UXtNJkYg",
		"email":          email,
		"email_verified": true,
		"groups": []string{
			"admins",
		},
		"name": "Rene Dekker",
	}
	payloadJson, _ := json.Marshal(payload)
	payloadStr := base64.RawURLEncoding.EncodeToString(payloadJson)
	return fmt.Sprintf("Bearer %s.%s.%s", hdrhdr, payloadStr, "e30"), payloadJson
}
