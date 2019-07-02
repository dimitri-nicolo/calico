// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package collector

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/google/gopacket/layers"
)

type DNSMeta struct {
	ClientMeta   EndpointMetadata       `json:"clientMeta"`
	Question     DNSName                `json:"dnsQuestion"`
	ResponseCode layers.DNSResponseCode `json:"dnsResponseCode"`
	RRSetsString string                 `json:"-"`
}

type DNSSpec struct {
	RRSets       DNSRRSets
	Servers      map[EndpointMetadata]DNSLabels
	ClientLabels DNSLabels
	DNSStats
}

type DNSSpecEncoded struct {
	RRSets       DNSRRSets   `json:"rrsets"`
	Servers      []DNSServer `json:"servers"`
	ClientLabels DNSLabels   `json:"clientLabels"`
	DNSStats
}

func (a *DNSSpec) Merge(b DNSSpec) {
	for e, l := range b.Servers {
		a.Servers[e] = l
	}
	a.ClientLabels = b.ClientLabels
	a.Count += b.Count
}

func (a *DNSSpec) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Encode())
}

func (a *DNSSpec) Encode() *DNSSpecEncoded {
	b := &DNSSpecEncoded{
		RRSets:       a.RRSets,
		ClientLabels: a.ClientLabels,
		DNSStats:     a.DNSStats,
	}
	for e, l := range a.Servers {
		b.Servers = append(b.Servers, DNSServer{e, l})
	}
	return b
}

func (a *DNSSpec) UnmarshalJSON(data []byte) error {
	var b DNSSpecEncoded
	err := json.Unmarshal(data, &b)
	if err != nil {
		return err
	}
	*a = *b.Decode()
	return nil
}

func (e *DNSSpecEncoded) Decode() *DNSSpec {
	a := &DNSSpec{
		RRSets:       e.RRSets,
		ClientLabels: e.ClientLabels,
		Servers:      make(map[EndpointMetadata]DNSLabels),
		DNSStats: DNSStats{
			Count: e.Count,
		},
	}
	for _, s := range e.Servers {
		a.Servers[s.EndpointMetadata] = s.Labels
	}
	return a
}

type DNSName struct {
	Name  string
	Class DNSClass
	Type  DNSType
}

type dnsNameEncoded struct {
	Name  string      `json:"name"`
	Class interface{} `json:"class"`
	Type  interface{} `json:"type"`
}

func (d DNSName) MarshalJSON() ([]byte, error) {
	n := d.encodeDNSName()

	return json.Marshal(&n)
}

func (d DNSName) encodeDNSName() dnsNameEncoded {
	n := dnsNameEncoded{
		d.Name,
		layers.DNSClass(d.Class).String(),
		layers.DNSType(d.Type).String(),
	}
	if n.Class == "Unknown" {
		n.Class = uint(d.Class)
	}
	if n.Type == "Unknown" {
		n.Type = uint(d.Type)
	}
	return n
}

func (d *DNSName) UnmarshalJSON(data []byte) error {
	var n dnsNameEncoded
	err := json.Unmarshal(data, &n)
	if err != nil {
		return err
	}

	return d.decodeDNSName(n)
}

func (d *DNSName) decodeDNSName(n dnsNameEncoded) error {
	d.Name = n.Name

	switch v := n.Class.(type) {
	case string:
		switch v {
		// Brittle. I don't know how to do this with reflection but that would be the best way to do it.
		case "IN":
			d.Class = DNSClass(layers.DNSClassIN)
		case "CS":
			d.Class = DNSClass(layers.DNSClassCS)
		case "CH":
			d.Class = DNSClass(layers.DNSClassCH)
		case "HS":
			d.Class = DNSClass(layers.DNSClassHS)
		case "ANY":
			d.Class = DNSClass(layers.DNSClassAny)
		default:
			return fmt.Errorf("Unknown class: %q", v)
		}
	case int:
		d.Class = DNSClass(v)
	default:
		return fmt.Errorf("Unknown class: %v", v)
	}

	switch v := n.Type.(type) {
	case string:
		switch v {
		// Brittle. I don't know how to do this with reflection but that would be the best way to do it.
		case "A":
			d.Type = DNSType(layers.DNSTypeA)
		case "NS":
			d.Type = DNSType(layers.DNSTypeNS)
		case "MD":
			d.Type = DNSType(layers.DNSTypeMD)
		case "MF":
			d.Type = DNSType(layers.DNSTypeMF)
		case "CNAME":
			d.Type = DNSType(layers.DNSTypeCNAME)
		case "SOA":
			d.Type = DNSType(layers.DNSTypeSOA)
		case "MB":
			d.Type = DNSType(layers.DNSTypeMB)
		case "MG":
			d.Type = DNSType(layers.DNSTypeMG)
		case "MR":
			d.Type = DNSType(layers.DNSTypeMR)
		case "NULL":
			d.Type = DNSType(layers.DNSTypeNULL)
		case "WKS":
			d.Type = DNSType(layers.DNSTypeWKS)
		case "PTR":
			d.Type = DNSType(layers.DNSTypePTR)
		case "HINFO":
			d.Type = DNSType(layers.DNSTypeHINFO)
		case "MINFO":
			d.Type = DNSType(layers.DNSTypeMINFO)
		case "MX":
			d.Type = DNSType(layers.DNSTypeMX)
		case "TXT":
			d.Type = DNSType(layers.DNSTypeTXT)
		case "AAAA":
			d.Type = DNSType(layers.DNSTypeAAAA)
		case "SRV":
			d.Type = DNSType(layers.DNSTypeSRV)
		default:
			return fmt.Errorf("Unknown type: %q", v)
		}
	case int:
		d.Type = DNSType(v)
	default:
		return fmt.Errorf("Unknown type: %v", v)
	}

	return nil
}

func (d DNSName) String() string {
	return fmt.Sprintf("%s %s %s", d.Name, d.Class, d.Type)
}

func (a DNSName) Less(b DNSName) bool {
	reverse := func(s []string) []string {
		for i := 0; i < len(s)/2; i++ {
			j := len(s) - i - 1
			s[i], s[j] = s[j], s[i]
		}
		return s
	}

	l := strings.Join(reverse(strings.Split(a.Name, ".")), ".")
	r := strings.Join(reverse(strings.Split(b.Name, ".")), ".")

	c := strings.Compare(l, r)
	switch {
	case c < 0:
		return true
	case c > 0:
		return false
	}

	switch {
	case a.Class < b.Class:
		return true
	case a.Class > b.Class:
		return false
	}

	return a.Type < b.Type
}

type DNSClass layers.DNSClass

func (d DNSClass) String() string {
	c := layers.DNSClass(d).String()
	if c != "Unknown" {
		return c
	}
	return fmt.Sprintf("#%d", d)
}

type DNSType layers.DNSType

func (d DNSType) String() string {
	t := layers.DNSType(d).String()
	if t != "Unknown" {
		return t
	}
	return fmt.Sprintf("#%d", d)
}

type DNSNames []DNSName

func (d DNSNames) Len() int {
	return len(d)
}

func (d DNSNames) Less(i, j int) bool {
	return d[i].Less(d[j])
}

func (d DNSNames) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

type DNSRRSets map[DNSName]DNSRDatas

func (d DNSRRSets) String() string {
	var s []string
	var names DNSNames

	for n, _ := range d {
		names = append(names, n)
	}
	sort.Sort(names)

	for _, n := range names {
		for _, r := range d[n] {
			s = append(s, fmt.Sprintf("%s %s", n, r))
		}
	}
	return strings.Join(s, "\n")
}

// Add inserts a DNSRData into the appropriate DNSRDatas in sorted order
func (d DNSRRSets) Add(name DNSName, rdata DNSRData) {
	index := sort.Search(len(d[name]), func(i int) bool { return !d[name][i].Less(rdata) })
	d[name] = append(d[name], DNSRData{})
	copy(d[name][index+1:], d[name][index:])
	d[name][index] = rdata
}

type dnsRRSetsEncoded struct {
	dnsNameEncoded
	RData DNSRDatas `json:"rdata"`
}

func (d DNSRRSets) MarshalJSON() ([]byte, error) {
	var r []dnsRRSetsEncoded
	for name, rdatas := range d {
		r = append(r, dnsRRSetsEncoded{name.encodeDNSName(), rdatas})
	}

	return json.Marshal(r)
}

func (d *DNSRRSets) UnmarshalJSON(data []byte) error {
	var r []dnsRRSetsEncoded
	err := json.Unmarshal(data, &r)
	if err != nil {
		return err
	}

	*d = make(DNSRRSets)
	for _, a := range r {
		n := DNSName{}
		err = n.decodeDNSName(a.dnsNameEncoded)
		if err != nil {
			return err
		}
		(*d)[n] = a.RData
	}
	return nil
}

type DNSRDatas []DNSRData

func (d DNSRDatas) Len() int {
	return len(d)
}

func (d DNSRDatas) Less(i, j int) bool {
	return d[i].Less(d[j])
}

func (d DNSRDatas) Swap(i, j int) {
	d[j], d[i] = d[i], d[j]
}

type DNSRData struct {
	Raw     []byte
	Decoded interface{}
}

func (a DNSRData) Less(b DNSRData) bool {
	return bytes.Compare(a.Raw, b.Raw) < 0
}

func (d DNSRData) String() string {
	switch v := d.Decoded.(type) {
	case net.IP:
		return v.String()
	case string:
		return v
	case []byte:
		return base64.StdEncoding.EncodeToString(v)
	case [][]byte:
		// This might not be the right thing to do here. It depends on how gopacket interprets multiple
		// TXT records.
		return string(bytes.Join(v, []byte{}))
	case layers.DNSSOA:
		return fmt.Sprintf("%s %s %d %d %d %d %d", v.MName, v.RName, v.Serial, v.Refresh, v.Retry, v.Expire, v.Minimum)
	case layers.DNSSRV:
		return fmt.Sprintf("%d %d %d %s", v.Priority, v.Weight, v.Port, v.Name)
	case layers.DNSMX:
		return fmt.Sprintf("%d %s", v.Preference, v.Name)
	default:
		return fmt.Sprintf("%#v", d.Decoded)
	}
}

func (d DNSRData) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *DNSRData) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	d.Raw = nil
	d.Decoded = s

	return nil
}

type DNSServer struct {
	EndpointMetadata
	Labels DNSLabels `json:"labels,omitempty"`
}

type DNSLabels map[string]string

type DNSStats struct {
	Count uint `json:"count"`
}

type DNSData struct {
	DNSMeta
	DNSSpec
}

type DNSLog struct {
	StartTime, EndTime time.Time
	DNSMeta
	DNSSpecEncoded
}

func (d *DNSData) ToDNSLog(startTime, endTime time.Time, includeLabels bool) *DNSLog {
	dl := &DNSLog{
		StartTime:      startTime,
		EndTime:        endTime,
		DNSMeta:        d.DNSMeta,
		DNSSpecEncoded: *d.DNSSpec.Encode(),
	}

	return dl
}
