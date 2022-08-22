// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package cache

import (
	"net"
	"reflect"

	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/deep-packet-inspection/pkg/weputils"
	bapi "github.com/projectcalico/calico/libcalico-go/lib/backend/api"
	"github.com/projectcalico/calico/libcalico-go/lib/backend/model"
	"github.com/projectcalico/calico/libcalico-go/lib/set"
)

type WEPCache interface {
	// Update caches all WEP key and WEP Data like pod name, namespace, interface name and IPs.
	// Snort generated alert only has source and destination IP, these cached details are used to populate the source or
	// destination name and namespace from the IP address while populating security event.
	Update(updateType bapi.UpdateType, wepKVPair model.KVPair)

	// Get returns the pod name and namespace for the give ip address.
	Get(ip string) (ok bool, podName, namespace string)

	// Clears the cache.
	Flush()
}

func NewWEPCache() WEPCache {
	return &wepCache{
		wepKeyToWEPData: make(map[model.WorkloadEndpointKey]wepData),
		ipToWEPKey:      make(map[string]model.WorkloadEndpointKey),
	}
}

// wepCache implements interface WEPCache
type wepCache struct {
	// ipToWEPKey maps IP to WEP key, it is in turn used to map WEP Key to WEP Data needed for processing the alert file content.
	ipToWEPKey map[string]model.WorkloadEndpointKey

	// wepKeyToWEPData is used to cache pod name, namespace, IPs to WEP Key.
	wepKeyToWEPData map[model.WorkloadEndpointKey]wepData
}

type wepData struct {
	ipList  set.Set[string]
	ns      string
	podName string
}

// Get returns the pod name and namespace for the give ip address.
func (r *wepCache) Get(ip string) (ok bool, podName, namespace string) {
	if wepKey, ok := r.ipToWEPKey[ip]; ok {
		if d, ok := r.wepKeyToWEPData[wepKey]; ok {
			return ok, d.podName, d.ns
		}
	}
	return false, "", ""
}

// Update caches all WEP key and WEP Data like pod name, namespace, interface name and IPs.
// Snort generated alert only has source and destination IP, these cached details are used to populate the source or
// destination name and namespace from the IP address while populating security event.
func (r *wepCache) Update(updateType bapi.UpdateType, wepKVPair model.KVPair) {
	log.WithField("WEP", wepKVPair.Key).Debugf("Updating alert WEP cache.")
	switch updateType {
	case bapi.UpdateTypeKVNew, bapi.UpdateTypeKVUpdated:
		if wepKey, ok := wepKVPair.Key.(model.WorkloadEndpointKey); ok {

			ns, name, err := weputils.ExtractNamespaceAndNameFromWepName(wepKey.WorkloadID)
			if err != nil {
				log.WithError(err).Errorf("Failed to extract namespace and pod name from %s", wepKey.WorkloadID)
				return
			}

			newIPs := extractIPsFromWorkloadEndpoint(wepKVPair.Value.(*model.WorkloadEndpoint))
			data, ok := r.wepKeyToWEPData[wepKey]
			if ok {
				// Remove the old IPs that are no longer in the WEP
				oldIPList := data.ipList
				oldIPList.Iter(func(item string) error {
					for i := range newIPs {
						if !reflect.DeepEqual(newIPs[i], item) {
							delete(r.ipToWEPKey, item)
						}
					}
					return nil
				})
			} else {
				data = wepData{
					ns:      ns,
					podName: name,
				}
			}

			// Cache all the IPs in WEP
			data.ipList = set.New[string]()
			for i := range newIPs {
				ip := net.IP(newIPs[i][:16])
				data.ipList.Add(ip.String())
				r.ipToWEPKey[ip.String()] = wepKey
			}

			r.wepKeyToWEPData[wepKey] = data
		}
	case bapi.UpdateTypeKVDeleted:
		if wepKey, ok := wepKVPair.Key.(model.WorkloadEndpointKey); ok {
			oldWEPData, ok := r.wepKeyToWEPData[wepKey]
			if ok {
				oldWEPData.ipList.Iter(func(item string) error {
					delete(r.ipToWEPKey, item)
					return nil
				})
				delete(r.wepKeyToWEPData, wepKey)
			}
		}
	}
}

func (r *wepCache) Flush() {
	r.ipToWEPKey = nil
	r.wepKeyToWEPData = nil
}

// ExtractIPsFromWorkloadEndpoint converts the IPv[46]Nets fields of the WorkloadEndpoint into
// [16]bytes. It ignores any prefix length.
// This logic is copied from felix
func extractIPsFromWorkloadEndpoint(endpoint *model.WorkloadEndpoint) [][16]byte {
	v4Nets := endpoint.IPv4Nets
	v6Nets := endpoint.IPv6Nets
	combined := make([][16]byte, 0, len(v4Nets)+len(v6Nets))
	for _, addr := range v4Nets {
		var addrB [16]byte
		copy(addrB[:], addr.IP.To16()[:16])
		combined = append(combined, addrB)
	}
	for _, addr := range v6Nets {
		var addrB [16]byte
		copy(addrB[:], addr.IP.To16()[:16])
		combined = append(combined, addrB)
	}
	return combined
}
