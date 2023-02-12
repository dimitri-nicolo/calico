// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package v1

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/projectcalico/calico/libcalico-go/lib/json"

	"github.com/google/gopacket/layers"
	"golang.org/x/net/idna"
)

// DNSLogParams define querying parameters to retrieve DNS logs
type DNSLogParams struct {
	QueryParams        `json:",inline" validate:"required"`
	LogSelectionParams `json:",inline"`
}

type DNSLog struct {
	StartTime       time.Time         `json:"start_time"`
	EndTime         time.Time         `json:"end_time"`
	Type            DNSLogType        `json:"type"`
	Count           uint              `json:"count"`
	ClientName      string            `json:"client_name"`
	ClientNameAggr  string            `json:"client_name_aggr"`
	ClientNamespace string            `json:"client_namespace"`
	ClientIP        *net.IP           `json:"client_ip"`
	ClientLabels    map[string]string `json:"client_labels"`
	Servers         []DNSServer       `json:"servers"`
	QName           QName             `json:"qname"`
	QClass          DNSClass          `json:"qclass"`
	QType           DNSType           `json:"qtype"`
	RCode           DNSResponseCode   `json:"rcode"`
	RRSets          DNSRRSets         `json:"rrsets"`
	Latency         DNSLatency        `json:"latency"`
	LatencyCount    int               `json:"latency_count"`
	LatencyMean     time.Duration     `json:"latency_mean"`
	LatencyMax      time.Duration     `json:"latency_max"`
}

type DNSLatency struct {
	// Number of successful latency measurements contributing to
	// the following mean and max.
	Count int `json:"count"`

	// Mean latency.
	Mean time.Duration `json:"mean"`

	// Max latency.
	Max time.Duration `json:"max"`
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
		aNameToUName(d.Name),
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

type DNSResponseCode layers.DNSResponseCode

func (d DNSResponseCode) String() string {
	if res, ok := dnsResponseCodeTable[d]; ok {
		return res
	}

	return fmt.Sprintf("#%d", d)
}

func (d DNSResponseCode) MarshalJSON() ([]byte, error) {
	if res, ok := dnsResponseCodeTable[d]; ok {
		return json.Marshal(&res)
	}

	i := uint(d)
	return json.Marshal(&i)
}

// Formatting from IANA DNS Parameters
var dnsResponseCodeTable = map[DNSResponseCode]string{
	0:  "NoError",
	1:  "FormErr",
	2:  "ServFail",
	3:  "NXDomain",
	4:  "NotImp",
	5:  "Refused",
	6:  "YXDomain",
	7:  "YXRRSet",
	8:  "NXRRSet",
	9:  "NotAuth",
	10: "NotZone",
	11: "DSOTYPENI",
	16: "BADSIG",
	17: "BADKEY",
	18: "BADTIME",
	19: "BADMODE",
	20: "BADNAME",
	21: "BADALG",
	22: "BADTRUNC",
	23: "BADCOOKIE",
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

	for n := range d {
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

// IDNAString is like String() but decodes international domain names to unicode
func (d DNSRData) IDNAString() string {
	switch v := d.Decoded.(type) {
	case net.IP:
		return v.String()
	case string:
		return aNameToUName(v)
	case []byte:
		return base64.StdEncoding.EncodeToString(v)
	case [][]byte:
		// This might not be the right thing to do here. It depends on how gopacket interprets multiple
		// TXT records.
		return string(bytes.Join(v, []byte{}))
	case layers.DNSSOA:
		return fmt.Sprintf("%s %s %d %d %d %d %d",
			aNameToUName(string(v.MName)), aNameToUName(string(v.RName)), v.Serial, v.Refresh, v.Retry, v.Expire, v.Minimum)
	case layers.DNSSRV:
		return fmt.Sprintf("%d %d %d %s", v.Priority, v.Weight, v.Port, aNameToUName(string(v.Name)))
	case layers.DNSMX:
		return fmt.Sprintf("%d %s", v.Preference, aNameToUName(string(v.Name)))
	default:
		return fmt.Sprintf("%#v", d.Decoded)
	}
}

func (d DNSRData) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.IDNAString())
}

type DNSServer struct {
	Endpoint
	IP     net.IP
	Labels map[string]string
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
		IP:        d.IP.String(),
	})
}

type DNSStats struct {
	Count uint `json:"count"`
}

type DNSLogType string

const (
	DNSLogTypeLog      DNSLogType = "log"
	DNSLogTypeUnlogged DNSLogType = "unlogged"
)

type QName string

func (q QName) MarshalJSON() ([]byte, error) {
	u := aNameToUName(string(q))
	return json.Marshal(u)
}

var (
	idnaProfile *idna.Profile
	ipOnce      sync.Once
)

// aNameToUName takes an "A-Name" (ASCII encoded) and converts it to a "U-Name"
// which is its unicode equivalent according to the International Domain Names
// for Applications (IDNA) spec (https://tools.ietf.org/html/rfc5891)
func aNameToUName(aname string) string {
	ipOnce.Do(func() {
		idnaProfile = idna.New()
	})
	u, err := idnaProfile.ToUnicode(aname)
	if err != nil {
		// If there was some problem converting, just return the name as we
		// encountered it in the DNS protocol
		return aname
	}
	return u
}
