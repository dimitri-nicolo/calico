// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package authentication_test

import (
	"net/http"
	"testing"

	"k8s.io/apiserver/pkg/endpoints/request"

	"github.com/projectcalico/apiserver/pkg/authentication"
)

const (
	validHeader        = "Bearer: valid"
	invalidHeader      = "Bearer: invalid"
	trigger500Header   = "Bearer 500"
	unauthorizedHeader = "Bearer: unauthorized"

	validUsername           = "jane"
	group                   = "system:authenticated"
	tokenReviewResponse     = `{"kind":"Status","apiVersion":"v1","metadata":{},"status":"Failure","message":"Unauthorized","reason":"Unauthorized","code":401}`
	forbiddenReviewResponse = `{"kind":"Status","apiVersion":"v1","metadata":{},"status":"Failure","message":"authenticationreviews.projectcalico.org is forbidden: User \"system:serviceaccount:tigera-compliance:unauthorized\" cannot create resource \"authenticationreviews\" in API group \"projectcalico.org\" at the cluster scope","reason":"Forbidden","details":{"group":"projectcalico.org","kind":"authenticationreviews"},"code":403}`
)

// Test the authenticator logic, by using a fake authenticator client.
func TestAuthenticate(t *testing.T) {
	authenticator := authentication.NewFakeAuthenticator()

	// Test a valid user
	authenticator.AddValidApiResponse(validHeader, validUsername, []string{group})
	authenticationReview, statusCode, err := authenticator.Authenticate(validHeader)
	if err != nil || authenticationReview.GetName() != validUsername || authenticationReview.GetGroups()[0] != group {
		t.Fatalf("test failure for header: %s. statusCode=%d, authenticationReview=%v, group=%v",
			validHeader, statusCode, authenticationReview, authenticationReview.GetGroups()[0])
	}

	// Test a user that is not authenticated. Verify that the authenticator does not trip over a response with
	// json that does represent an authentication review.
	authenticator.AddErrorAPIServerResponse(invalidHeader, []byte(tokenReviewResponse), http.StatusUnauthorized)
	authenticationReview, statusCode, err = authenticator.Authenticate(invalidHeader)
	if authenticationReview != nil || err == nil || statusCode != http.StatusUnauthorized {
		t.Fatalf("test failure for header: %s. statusCode=%d, authenticationReview=%v",
			invalidHeader, statusCode, authenticationReview)
	}

	// Test a user that does not have the rbac privileges to perform an authenticationreview.
	authenticator.AddErrorAPIServerResponse(unauthorizedHeader, []byte(forbiddenReviewResponse), http.StatusForbidden)
	authenticationReview, statusCode, err = authenticator.Authenticate(unauthorizedHeader)
	if authenticationReview != nil || err == nil || statusCode != http.StatusForbidden {
		t.Fatalf("test failure for header: %s. statusCode=%d, authenticationReview=%v",
			unauthorizedHeader, statusCode, authenticationReview)
	}

	// Verify that an unexpected tigera-server response results in a 500
	authenticator.AddErrorAPIServerResponse(trigger500Header, []byte(tokenReviewResponse), http.StatusCreated)
	authenticationReview, statusCode, err = authenticator.Authenticate(trigger500Header)
	if authenticationReview != nil || err == nil || statusCode != http.StatusInternalServerError {
		t.Fatalf("test failure for header: %s. statusCode=%d, authenticationReview=%v",
			trigger500Header, statusCode, authenticationReview)
	}

}

// Test the utility function to authenticate a request and add user info.
func TestAuthenticateRequest(t *testing.T) {
	authenticator := authentication.NewFakeAuthenticator()

	// Test a valid user
	authenticator.AddValidApiResponse(validHeader, validUsername, []string{group})
	req := &http.Request{Header: http.Header{}}
	req.Header.Set(authentication.AuthorizationHeader, validHeader)
	req, statusCode, err := authentication.AuthenticateRequest(authenticator, req)
	info, ok := request.UserFrom(req.Context())
	if !ok || err != nil || info.GetName() != validUsername || info.GetGroups()[0] != group {
		t.Fatalf("test failure for header: %s. statusCode=%d, userInfo=%v, err=%v",
			validHeader, statusCode, info, err)
	}

	// Test a user that is not authenticated.
	authenticator.AddErrorAPIServerResponse(invalidHeader, []byte(tokenReviewResponse), http.StatusUnauthorized)
	req = &http.Request{Header: http.Header{}}
	req.Header.Set(authentication.AuthorizationHeader, invalidHeader)
	req, statusCode, err = authentication.AuthenticateRequest(authenticator, req)
	info, ok = request.UserFrom(req.Context())
	if info != nil || err == nil || statusCode != http.StatusUnauthorized {
		t.Fatalf("test failure for header: %s. statusCode=%d, userInfo=%v",
			invalidHeader, statusCode, info)
	}

	// Test a user that does not have the rbac privileges to perform an authenticationreview.
	authenticator.AddErrorAPIServerResponse(unauthorizedHeader, []byte(tokenReviewResponse), http.StatusForbidden)
	req = &http.Request{Header: http.Header{}}
	req.Header.Set(authentication.AuthorizationHeader, unauthorizedHeader)
	req, statusCode, err = authentication.AuthenticateRequest(authenticator, req)
	info, ok = request.UserFrom(req.Context())
	if info != nil || err == nil || statusCode != http.StatusForbidden {
		t.Fatalf("test failure for header: %s. statusCode=%d, userInfo=%v",
			unauthorizedHeader, statusCode, info)
	}

	// Verify that an unexpected tigera-server response results in a 500
	authenticator.AddErrorAPIServerResponse(trigger500Header, []byte(tokenReviewResponse), http.StatusCreated)
	req = &http.Request{Header: http.Header{}}
	req.Header.Set(authentication.AuthorizationHeader, trigger500Header)
	req, statusCode, err = authentication.AuthenticateRequest(authenticator, req)
	info, ok = request.UserFrom(req.Context())
	if info != nil || err == nil || statusCode != http.StatusInternalServerError {
		t.Fatalf("test failure for header: %s. statusCode=%d, userInfo=%v",
			trigger500Header, statusCode, info)
	}
}
