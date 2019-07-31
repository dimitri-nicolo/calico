// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	log "github.com/sirupsen/logrus"

	authzv1 "k8s.io/api/authorization/v1"
)

// The handler returned by this will add a ResourceAttribute to the context
// of the request based on the request.URL.Path. The ResourceAttribute
// is intended to be used with a SelfSubjectAccessReview or SubjectAccessReview
// to check if a user has access to the resource.
// Upon successful conversion/context update, the handler passed in will be
// called, otherwise the ResponseWriter will be updated with the appropriate
// status and a message with details.
func RequestToResource(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		name, err := getResourceNameFromReq(req)
		if err != nil {
			log.WithError(err).Debugf("Unable to convert request URL '%+v' to resource", req.URL)
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		h.ServeHTTP(w, req.WithContext(NewContextWithReviewResource(req.Context(), getResourceAttributes(name))))
	})
}

type contextKey int

const (
	ResourceAttributeKey contextKey = iota
	NonResourceAttributeKey
)

func NewContextWithReviewResource(
	ctx context.Context,
	ra *authzv1.ResourceAttributes,
) context.Context {
	return context.WithValue(ctx, ResourceAttributeKey, ra)
}

func NewContextWithReviewNonResource(
	ctx context.Context,
	ra *authzv1.NonResourceAttributes,
) context.Context {
	return context.WithValue(ctx, NonResourceAttributeKey, ra)
}

func FromContextGetReviewResource(ctx context.Context) (*authzv1.ResourceAttributes, bool) {
	ra, ok := ctx.Value(ResourceAttributeKey).(*authzv1.ResourceAttributes)
	return ra, ok
}

func FromContextGetReviewNonResource(ctx context.Context) (*authzv1.NonResourceAttributes, bool) {
	nra, ok := ctx.Value(NonResourceAttributeKey).(*authzv1.NonResourceAttributes)
	return nra, ok
}

func getResourceAttributes(resourceName string) *authzv1.ResourceAttributes {
	return &authzv1.ResourceAttributes{
		Verb:     "get",
		Group:    "lma.tigera.io",
		Resource: "index",
		Name:     resourceName,
	}
}

// getResourceNameFromReq parses the req.URL.Path, converting the indexes
// into the resource name used in RBAC.
// This implements the table located in
// https://docs.google.com/document/d/1wFrbjLydsdz0NfxVk-_iW7eqx4ZIZWfgj5SzcsRmTwo/edit#heading=h.pva3ex6ffysc
func getResourceNameFromReq(req *http.Request) (string, error) {
	if req.URL == nil {
		return "", fmt.Errorf("No URL in request")
	}
	queryResourceMap := map[string]string{
		"tigera_secure_ee_flows":      "flows",
		"tigera_secure_ee_audit_":     "audit*", // support both audit_*
		"tigera_secure_ee_audit":      "audit*", // and audit*
		"tigera_secure_ee_audit_ee":   "audit_ee",
		"tigera_secure_ee_audit_kube": "audit_kube",
		"tigera_secure_ee_events":     "events",
	}
	re := regexp.MustCompile(`/([_a-z]*)[.*].*/_search`)

	match := re.FindStringSubmatch(req.URL.Path)
	if len(match) != 2 {
		return "", fmt.Errorf("Invalid resource in path, '%s' had %d matches", req.URL.Path, len(match))
	}
	resource, ok := queryResourceMap[match[1]]
	if !ok {
		return "", fmt.Errorf("Invalid resource '%s' in path", match[1])
	}
	return resource, nil
}
