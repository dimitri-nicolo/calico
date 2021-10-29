// package net contains utility operations for network datastructures
package net

import (
	"crypto/sha1"
	gonet "net"
)

type MACBuilder struct {
	cacheMACByNodename map[string]gonet.HardwareAddr
}

func NewMACBuilder() *MACBuilder {
	return &MACBuilder{
		make(map[string]gonet.HardwareAddr),
	}
}

// GenerateMAC creates unique MAC addresses based off nodenames. Based on https://github.com/tigera/felix-private/blob/b3b7e7a95889eecd923fda9d3a68e74c07e670d4/calc/vxlan_resolver.go#L276
func (m *MACBuilder) GenerateMAC(nodename string) (gonet.HardwareAddr, error) {
	// do a cache lookup before calc'ing
	mac, ok := m.cacheMACByNodename[nodename]
	if ok {
		return mac, nil
	}

	hasher := sha1.New()
	_, err := hasher.Write([]byte(nodename))
	if err != nil {
		return nil, err
	}

	sha := hasher.Sum(nil)
	hw := gonet.HardwareAddr(append([]byte("f"), sha[0:5]...))

	// cache result for performance
	m.cacheMACByNodename[nodename] = hw

	return hw, nil
}
