// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package collector

import (
	"net"

	log "github.com/sirupsen/logrus"
)

type IpKey [16]byte

func IpKeyFromNetIP(ip net.IP) IpKey {
	var ipk [16]byte
	copy(ipk[:], ip.To16())
	return ipk
}

func NetIPFromIpKey(ipk IpKey) net.IP {
	return net.IP(ipk[:16])
}

// boundedSet stores the count of unique IP addresses. Of the `totalCount` of
// IP addresses, it also stores up to `maxSize` number of unique IP addresses.
// Only `totalCount` is changed onces the maxSize items have been reached.
type boundedSet struct {
	// IP addresses tracked in this bounded set up to `maxSize`.
	ips map[IpKey]empty
	// The maximum number of IP address values that this boundedSet will track.
	// Any Adds beyond maxSize will only increment the `totalCount` field.
	maxSize int
	// `totalCount` trackes the total number of unique IP addresses tracked
	// in this boundedSet. Of `totalCount` IP addresses, up to `maxSize` IP
	// address values are available in the `ips` map.
	totalCount *Counter
}

// NewBoundedSet creates a boundedSet which will store a maximum of `maxSize` items.
func NewBoundedSet(maxSize int) *boundedSet {
	return &boundedSet{
		ips:        make(map[IpKey]empty, maxSize),
		maxSize:    maxSize,
		totalCount: NewCounter(0),
	}
}

// NewBoundedSetFromSlice creates a boundedSet from the given slice of IP addresses.
func NewBoundedSetFromSlice(maxSize int, items []net.IP) *boundedSet {
	bs := NewBoundedSet(maxSize)
	for _, item := range items {
		bs.Add(item)
	}
	return bs
}

// NewBoundedSetFromSliceWithTotalCount creates a boundedSet from the given slice of IP addresses
// and also sets the `totalCount` of items. This is useful for pre-creating a boundedSet from an
// existing set of data.
func NewBoundedSetFromSliceWithTotalCount(maxSize int, items []net.IP, totalCount int) *boundedSet {
	if totalCount < len(items) {
		log.WithFields(log.Fields{"totalCount": totalCount, "numItems": len(items)}).Error("totalCount less than number of items")
		return nil
	}
	bs := NewBoundedSet(maxSize)
	for _, item := range items {
		bs.Add(item)
		totalCount--
	}
	bs.totalCount.Increase(totalCount)
	return bs
}

func (set *boundedSet) MaxSize() int {
	return set.maxSize
}

// TotalCount returns the totalCount of the boundedSet.
func (set *boundedSet) TotalCount() int {
	return set.totalCount.Absolute()
}

// TotalCountDelta returns the last batch of increase in totalCount of the boundedSet since the last time
// this method was called.
func (set *boundedSet) TotalCountDelta() int {
	tc := set.totalCount.Delta()
	return tc
}

func (set *boundedSet) ResetDeltaCount() {
	set.totalCount.ResetDelta()
}

// Add an IP address to the set. If the item being added is the "maxSize + 1"-th item
// then only the totalCount is changed and the item is not stored in the set.
func (set *boundedSet) Add(ip net.IP) {
	// If we've already tracked this IP, then no-op here.
	if set.Contains(ip) {
		return
	}
	if len(set.ips) < set.maxSize {
		set.ips[IpKeyFromNetIP(ip)] = emptyValue
	}
	set.totalCount.Increase(1)
}

// IncreaseTotalCount increases the total count by deltaCount.
func (set *boundedSet) IncreaseTotalCount(deltaCount int) {
	set.totalCount.Increase(deltaCount)
}

// Contains checks if IP address is present in the boundedSet. If the IP address
// was added after `maxSize` elements were already present then Contains will
// return false.
func (set *boundedSet) Contains(ip net.IP) bool {
	_, present := set.ips[IpKeyFromNetIP(ip)]
	return present
}

// Copy instantiates and creates a new boundedSet of the same maxSize and items
// from the source boundedSet.
func (set *boundedSet) Copy() *boundedSet {
	bs := NewBoundedSet(set.maxSize)
	bsTotalCount := set.TotalCount()
	for ipk := range set.ips {
		bs.Add(NetIPFromIpKey(ipk))
		bsTotalCount--
	}
	bs.IncreaseTotalCount(bsTotalCount)
	return bs
}

// Reset the boundedSet keeping the maxSize the same.
func (set *boundedSet) Reset() {
	set.ips = make(map[IpKey]empty, set.maxSize)
	set.totalCount.Reset()
}

// Combine items from the provided boundedSet into this boundedSet. If the items
// in the combined boundedSet is greater than `maxSize` then only the `totalCount`
// us incremented.
func (set *boundedSet) Combine(bs *boundedSet) {
	bsCount := bs.totalCount.Absolute()
	for ipk := range bs.ips {
		set.Add(NetIPFromIpKey(ipk))
		bsCount--
	}
	set.totalCount.Increase(bsCount)
}

// ToIPSlice returns a slice of the IP addresses tracked in the boundedSet.
func (set *boundedSet) ToIPSlice() []net.IP {
	slips := make([]net.IP, 0, len(set.ips))
	for ipk := range set.ips {
		slips = append(slips, NetIPFromIpKey(ipk))
	}
	return slips
}
