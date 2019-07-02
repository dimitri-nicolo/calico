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

	"github.com/projectcalico/felix/calc"
)

type DNSUpdate struct {
	ClientIP net.IP
	ClientEP *calc.EndpointData
	ServerIP net.IP
	ServerEP *calc.EndpointData
	DNS      *layers.DNS
}

type EndpointMetadataWithIP struct {
	EndpointMetadata
	IP string
}

type DNSMeta struct {
	ClientMeta   EndpointMetadataWithIP
	Question     DNSName
	ResponseCode layers.DNSResponseCode
	RRSetsString string
}

type DNSSpec struct {
	RRSets       DNSRRSets
	Servers      map[EndpointMetadataWithIP]DNSLabels
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

func (d DNSClass) MarshalJSON() ([]byte, error) {
	s := d.String()
	return json.Marshal(&s)
}

type DNSType layers.DNSType

func (d DNSType) String() string {
	t := layers.DNSType(d).String()
	if t != "Unknown" {
		return t
	}
	return fmt.Sprintf("#%d", d)
}

func (d DNSType) MarshalJSON() ([]byte, error) {
	s := d.String()
	return json.Marshal(&s)
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

type DNSServer struct {
	EndpointMetadataWithIP
	Labels DNSLabels
}

type dnsServerEncoded struct {
	Name      string `json:"name"`
	NameAggr  string `json:"name_aggr"`
	Namespace string `json:"namespace"`
	IP        string `json:"ip"`
}

func (d DNSServer) MarshalJSON() ([]byte, error) {
	return json.Marshal(&dnsServerEncoded{
		Name:      d.Name,
		NameAggr:  d.AggregatedName,
		Namespace: d.Namespace,
		IP:        d.IP,
	})
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
	StartTime       time.Time         `json:"start_time"`
	EndTime         time.Time         `json:"end_time"`
	Count           uint              `json:"count"`
	ClientName      string            `json:"client_name"`
	ClientNameAggr  string            `json:"client_name_aggr"`
	ClientNamespace string            `json:"client_namespace"`
	ClientIP        string            `json:"client_ip"`
	ClientLabels    map[string]string `json:"client_labels"`
	Servers         []DNSServer       `json:"servers"`
	QName           string            `json:"qname"`
	QClass          DNSClass          `json:"qclass"`
	QType           DNSType           `json:"qtype"`
	RCode           string            `json:"rcode"`
	RRSets          DNSRRSets         `json:"rrsets"`
}

func (d *DNSData) ToDNSLog(startTime, endTime time.Time, includeLabels bool) *DNSLog {
	e := d.DNSSpec.Encode()

	return &DNSLog{
		StartTime:       startTime,
		EndTime:         endTime,
		Count:           d.Count,
		ClientName:      d.ClientMeta.Name,
		ClientNameAggr:  d.ClientMeta.AggregatedName,
		ClientNamespace: d.ClientMeta.Namespace,
		ClientIP:        d.ClientMeta.IP,
		ClientLabels:    d.ClientLabels,
		Servers:         e.Servers,
		QName:           d.Question.Name,
		QClass:          d.Question.Class,
		QType:           d.Question.Type,
		RCode:           d.ResponseCode.String(),
		RRSets:          e.RRSets,
	}
}
