// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package collector

import (
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

func NewL7MetaSpecFromUpdate(update L7Update, sk L7SvcAggregationKind, uk L7URLAggregationKind, ek L7ErrAggregationKind) (L7Meta, L7Spec, error) {
	meta := L7Meta{
		ResponseCode:     update.ResponseCode,
		Method:           update.Method,
		Domain:           update.Domain,
		Path:             update.Path,
		UserAgent:        update.UserAgent,
		Type:             update.Type,
		ServiceName:      update.ServiceName,
		ServiceNamespace: update.ServiceNamespace,
		ServicePort:      update.ServicePort,
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
	meta.DstNamespace = dstMeta.Namespace
	meta.SrcType = srcMeta.Type
	meta.DstType = dstMeta.Type

	// If we have a service and the service namespace has not been set, default it to the destination namespace.
	if meta.ServiceName != "" && meta.ServiceNamespace == "" {
		meta.ServiceNamespace = dstMeta.Namespace
	}

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

func getAddressAndPort(domain string) (string, int) {
	parts := strings.Split(domain, ":")
	if len(parts) == 1 {
		// There is no port specified
		return parts[0], 0
	}

	if len(parts) == 2 {
		// There is a port specified
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			log.WithError(err).Error("Failed to parse port from L7 domain field")
			return "", 0
		}
		return parts[0], port
	}

	// If the domain is weird and has multiple ":" characters, then return nothing.
	return "", 0
}

// Extracts the Kubernetes service name if the address matches a Kubernetes service.
func extractK8sServiceNameAndNamespace(addr string) (string, string) {
	// Kubernetes service names can be in the format: <name>.<namespace>.svc.<cluster-domain>.<local>
	if parts := strings.Split(addr, "."); len(parts) > 4 && parts[len(parts)-3] == "svc" {
		return strings.Join(parts[:len(parts)-4], "."), parts[len(parts)-4]
	}

	// Kubernetes service names can be in the format: <name>.svc.<cluster-domain>.<local>
	if parts := strings.Split(addr, "."); len(parts) > 3 && parts[len(parts)-3] == "svc" {
		return strings.Join(parts[:len(parts)-3], "."), ""
	}

	// Not a valid Kubernetes service name
	return "", ""
}
