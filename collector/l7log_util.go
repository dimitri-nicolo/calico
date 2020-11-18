// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package collector

import log "github.com/sirupsen/logrus"

func NewL7MetaSpecFromUpdate(update L7Update, sk L7SvcAggregationKind, uk L7URLAggregationKind, ek L7ErrAggregationKind) (L7Meta, L7Spec, error) {
	meta := L7Meta{
		ResponseCode: update.ResponseCode,
		Method:       update.Method,
		Domain:       update.Domain,
		Path:         update.Path,
		UserAgent:    update.UserAgent,
		Type:         update.Type,
	}

	// Get source endpoint metadata
	srcMeta, err := getFlowLogEndpointMetadata(update.SrcEp, update.Tuple.src)
	if err != nil {
		log.WithError(err).Errorf("Failed to extract metadata for source %v", update.SrcEp)
	}

	// Get destination endpoint metadata
	dstMeta, err := getFlowLogEndpointMetadata(update.DstEp, update.Tuple.dst)
	if err != nil {
		log.WithError(err).Errorf("Failed to extract metadata for destination %v", update.DstEp)
	}

	meta.SrcNameAggr = srcMeta.AggregatedName
	meta.SrcNamespace = srcMeta.Namespace
	meta.DstNameAggr = dstMeta.AggregatedName
	meta.DstNamespace = srcMeta.Namespace
	meta.SrcType = srcMeta.Type
	meta.DstType = dstMeta.Type

	// TODO: Add service name when API is available
	// Service names need to be stored in the meta as a string of values separated by "," for proper keying to work.

	// TODO: Fix up the aggregation values
	/*
		if sk == L7DstSvcName {
			meta.DstService = flowLogFieldNotIncluded
		}

		u, err := url.Parse(update.Url)
		if err != nil {
			return meta, spec, err
		}
		switch uk {
		case L7URLQuery:
			// Trim URL of query params
			u.RawQuery = ""
			u.Fragment = ""
			meta.Url = u.String()
		case L7URLQueryPath:
			// Trim URL of query params and path
			u.RawQuery = ""
			u.Fragment = ""
			u.Path = ""
			meta.Url = u.String()
		case L7URLQueryPathBase:
			// Remove the URL entirely
			meta.Url = flowLogFieldNotIncluded
		}

		if ek == L7ErrorCode {
			meta.ResponseCode = flowLogFieldNotIncluded
		}
	*/

	spec := L7Spec{
		Duration:      update.Duration,
		DurationMax:   update.DurationMax,
		BytesReceived: update.BytesReceived,
		BytesSent:     update.BytesSent,
		Count:         update.Count,
	}

	return meta, spec, nil
}
