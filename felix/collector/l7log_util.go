// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package collector

import (
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

func NewL7MetaSpecFromUpdate(update L7Update, ak L7AggregationKind) (L7Meta, L7Spec, error) {
	meta := L7Meta{
		ResponseCode:     update.ResponseCode,
		Method:           update.Method,
		Domain:           update.Domain,
		Path:             update.Path,
		UserAgent:        update.UserAgent,
		Type:             update.Type,
		ServiceName:      update.ServiceName,
		ServiceNamespace: update.ServiceNamespace,
		ServicePortName:  update.ServicePortName,
		ServicePortNum:   update.ServicePortNum,
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
	meta.SourcePortNum = update.Tuple.l4Src

	meta.DestNameAggr = dstMeta.AggregatedName
	meta.DestNamespace = dstMeta.Namespace
	meta.DestPortNum = update.Tuple.l4Dst

	meta.SrcType = srcMeta.Type
	meta.DestType = dstMeta.Type

	// If we have a service and the service namespace has not been set, default it to the destination namespace.
	if meta.ServiceName != "" && meta.ServiceNamespace == "" {
		meta.ServiceNamespace = dstMeta.Namespace
	}

	// Handle aggregation and remove any unneeded values.
	if ak.HTTPHeader == L7HTTPHeaderInfoNone {
		meta.UserAgent = flowLogFieldNotIncluded
		meta.Type = flowLogFieldNotIncluded
	}

	if ak.HTTPMethod == L7HTTPMethodNone {
		meta.Method = flowLogFieldNotIncluded
	}

	if ak.Service == L7ServiceInfoNone {
		meta.ServiceName = flowLogFieldNotIncluded
		meta.ServiceNamespace = flowLogFieldNotIncluded
		meta.ServicePortName = flowLogFieldNotIncluded
		meta.ServicePortNum = 0
	}

	if ak.Destination == L7DestinationInfoNone {
		meta.DestNameAggr = flowLogFieldNotIncluded
		meta.DestNamespace = flowLogFieldNotIncluded
		meta.DestType = flowLogFieldNotIncluded
		meta.DestPortNum = 0
	}

	if ak.ResponseCode == L7ResponseCodeNone {
		meta.ResponseCode = flowLogFieldNotIncluded
	}

	switch ak.Source {
	case L7SourceInfoNone:
		meta.SrcNameAggr = flowLogFieldNotIncluded
		meta.SrcNamespace = flowLogFieldNotIncluded
		meta.SrcType = flowLogFieldNotIncluded
		meta.SourcePortNum = 0
	case L7SourceInfoNoPort:
		meta.SourcePortNum = 0
	}

	switch ak.TrimURL {
	case L7FullURL:
		// If the whole URL is specified, trim the path if required.
		if ak.NumURLPathParts >= 0 {
			// Remove the query portion of the URL
			path := strings.Split(update.Path, "?")[0]

			// Split the path into components and only grab the specified number of components.
			parts := strings.Split(path, "/")
			// Since the Path is expected to lead with "/", parts
			// will be 1 longer than the valid parts of the path.
			if len(parts) > ak.NumURLPathParts+1 {
				trimmed := []string{}
				i := 0
				for i < ak.NumURLPathParts+1 {
					trimmed = append(trimmed, parts[i])
					i++
				}
				parts = trimmed
			}
			meta.Path = strings.Join(parts, "/")
		}
	case L7URLWithoutQuery:
		// Trim path of query params
		meta.Path = strings.Split(meta.Path, "?")[0]
	case L7BaseURL:
		// Remove path
		meta.Path = flowLogFieldNotIncluded
	case L7URLNone:
		// Remove the URL entirely
		meta.Domain = flowLogFieldNotIncluded
		meta.Path = flowLogFieldNotIncluded
	}
	// once the processing is done eventually make sure URL length is manageable
	limitURLLength(&meta, ak.URLCharLimit)
	spec := L7Spec{
		Duration:      update.Duration,
		DurationMax:   update.DurationMax,
		BytesReceived: update.BytesReceived,
		BytesSent:     update.BytesSent,
		Latency:       update.Latency,
		Count:         update.Count,
	}

	return meta, spec, nil
}

func limitURLLength(meta *L7Meta, limit int) {
	// when the URL exceeds a configured limit, trim it down to manageable length
	if len(meta.Domain)+len(meta.Path) > limit {
		// path length that is permissible
		maxPath := limit - len(meta.Domain)
		if maxPath < 0 {
			// in this case we don't send the path at all and we limit domain to limit
			meta.Domain = meta.Domain[0:limit]
			meta.Path = flowLogFieldNotIncluded
		} else {
			meta.Path = meta.Path[0:maxPath]
		}
	}
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

	// Kubernetes service names can be in the format: <name>.<namespace>
	// Note that this check does not allow subdomains in the <name>
	if parts := strings.Split(addr, "."); len(parts) == 2 {
		return parts[0], parts[1]
	}

	// Not a valid Kubernetes service name
	return "", ""
}
