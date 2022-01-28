// Copyright (c) 2021 Tigera Inc. All rights reserved.
package elastic

func NewWAFConverter() ElasticQueryConverter {
	return &converter{basicAtomToElastic}
}
