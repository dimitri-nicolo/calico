// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package api

type Tier interface {
	GetName() string
	GetResource() Resource
	GetOrderedPolicies() []Policy
}
