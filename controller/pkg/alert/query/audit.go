// Copyright (c) 2019 Tigera Inc. All rights reserved.

package query

func NewAuditConverter() ElasticQueryConverter {
	return &converter{basicAtomToElastic}
}
