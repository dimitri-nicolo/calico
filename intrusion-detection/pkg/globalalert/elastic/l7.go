// Copyright (c) 2021 Tigera Inc. All rights reserved.
package elastic

func NewL7Converter() ElasticQueryConverter {
	return &converter{basicAtomToElastic}
}
