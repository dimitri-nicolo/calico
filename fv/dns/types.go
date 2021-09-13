// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package dns

type RecordIP struct {
	TTL uint32 `json:"ttl"`
	IP  string `json:"ip"`
}
