// Copyright (c) 2019 Tigera Inc. All rights reserved.

package elastic

func NewAuditConverter() ElasticQueryConverter {
	return &converter{basicAtomToElastic}
}
