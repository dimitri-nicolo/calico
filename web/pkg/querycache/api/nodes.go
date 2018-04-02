// Copyright (c) 2018 Tigera, Inc. All rights reserved.
package api

type Node interface {
	GetName() string
	GetResource() Resource
	GetEndpointCounts() EndpointCounts
}
