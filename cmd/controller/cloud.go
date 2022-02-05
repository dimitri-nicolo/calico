// Copyright (c) 2021 Tigera, Inc. All rights reserved.

//go:build tesla
// +build tesla

package main

import (
	"log"
	"net/url"
	"os"
	"regexp"

	"github.com/google/uuid"
)

var (
	// We assume that a tenant ID must obey the following syntax restrictions:
	//  - contain at most 63 characters
	//  - contain only lowercase alphanumeric characters or '-'
	//  - start with an alphanumeric character
	//  - end with an alphanumeric character
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-label-names
	tenantIDSyntax = regexp.MustCompile(`^[a-z0-9]([a-z0-9]|[a-z0-9\-]{0,61}[a-z0-9])?$`)
)

// ValidateEnvVars performs validation on environment variables that are specific to this variant (Cloud/Tesla).
func ValidateEnvVars() {
	// Including Tenant ID is optional for Cloud/Tesla. It should be enabled when using a multi-tenant setup.
	tenantID := os.Getenv("ELASTIC_INDEX_TENANT_ID")
	if !tenantIDSyntax.MatchString(tenantID) {
		log.Fatal("ELASTIC_INDEX_TENANT_ID must consist of only alpha-numeric chars (lowercase) or '-' and be at max 63 chars")
	}

	imageAssuranceEndpoint := os.Getenv("IMAGE_ASSURANCE_BAST_API_URL")
	if imageAssuranceEndpoint == "" {
		log.Fatal("IMAGE_ASSURANCE_BAST_API_URL can not be empty")
	}
	if _, err := url.Parse(imageAssuranceEndpoint); err != nil {
		log.Fatal("IMAGE_ASSURANCE_BAST_API_URL is not valid")
	}

	imageAssuranceCABundlePath := os.Getenv("IMAGE_ASSURANCE_CA_BUNDLE_PATH")
	if imageAssuranceCABundlePath == "" {
		log.Fatal("IMAGE_ASSURANCE_CA_BUNDLE_PATH can not be empty")
	}

	imageAssuranceOrgID := os.Getenv("IMAGE_ASSURANCE_ORGANIZATION_ID")
	if _, err := uuid.Parse(imageAssuranceOrgID); err != nil {
		log.Fatal("IMAGE_ASSURANCE_ORGANIZATION_ID must be a valid UUID")
	}
}
