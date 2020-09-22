// Copyright (c) 2018-2020 Tigera, Inc. All rights reserved.

package collector

import (
	"fmt"
	"net"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("FlowLog JSON serialization", func() {

	Describe("should set every field", func() {
		policies := FlowPolicies{
			"0|tier.policy|pass":                      emptyValue,
			"1|default.knp.default.default-deny|deny": emptyValue,
		}
		flowLog := FlowLog{
			StartTime: time.Now(),
			EndTime:   time.Now(),
			FlowMeta: FlowMeta{
				Tuple: Tuple{
					proto: 6,
					src:   [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
					dst:   [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
					l4Src: 345,
					l4Dst: 80,
				},
				SrcMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "test",
					Name:           "test",
					AggregatedName: "test-*",
				},
				DstMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "test",
					Name:           "test",
					AggregatedName: "test-*",
				},
				DstService: FlowService{
					Name:      "svc2",
					Namespace: "svc2-ns",
					Port:      "*",
				},
				Action:   "allow",
				Reporter: "src",
			},
			FlowLabels: FlowLabels{
				SrcLabels: map[string]string{"foo": "bar", "foo2": "bar2"},
				DstLabels: map[string]string{"foo": "bar", "foo2": "bar2"},
			},
			FlowPolicies: policies,
			FlowReportedStats: FlowReportedStats{
				PacketsIn:             1,
				PacketsOut:            2,
				BytesIn:               3,
				BytesOut:              4,
				NumFlowsStarted:       5,
				NumFlowsCompleted:     6,
				NumFlows:              7,
				HTTPRequestsAllowedIn: 8,
				HTTPRequestsDeniedIn:  9,
			},
			FlowExtras: FlowExtras{
				OriginalSourceIPs:    []net.IP{net.ParseIP("10.0.1.1")},
				NumOriginalSourceIPs: 1,
			},
		}

		out := toOutput(&flowLog)
		// Use reflection to loop over the fields and ensure they all have non
		// zero values
		oType := reflect.TypeOf(out)
		oVal := reflect.ValueOf(out)
		for i := 0; i < oType.NumField(); i++ {
			field := oType.Field(i)
			zeroVal := reflect.Zero(field.Type)
			actualVal := oVal.Field(i)
			It(fmt.Sprintf("should set %s", field.Name), func() {
				Expect(actualVal.Interface()).ToNot(Equal(zeroVal.Interface()))
			})
		}
	})

	Describe("should handle empty fields", func() {
		flowLog := FlowLog{
			StartTime: time.Now(),
			EndTime:   time.Now(),
			FlowMeta: FlowMeta{
				Tuple: Tuple{
					proto: 6,
					src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					l4Src: unsetIntField,
					l4Dst: 80,
				},
				SrcMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "test",
					Name:           "test",
					AggregatedName: "test-*",
				},
				DstMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "test",
					Name:           "test",
					AggregatedName: "test-*",
				},
				Action:   "allow",
				Reporter: "src",
			},
			FlowLabels: FlowLabels{
				SrcLabels: nil,
				DstLabels: nil,
			},
			FlowReportedStats: FlowReportedStats{
				PacketsIn:             1,
				PacketsOut:            2,
				BytesIn:               3,
				BytesOut:              4,
				NumFlowsStarted:       5,
				NumFlowsCompleted:     6,
				NumFlows:              7,
				HTTPRequestsAllowedIn: 8,
				HTTPRequestsDeniedIn:  9,
			},
			FlowExtras: FlowExtras{
				OriginalSourceIPs:    []net.IP{},
				NumOriginalSourceIPs: 0,
			},
		}

		out := toOutput(&flowLog)

		zeroFieldNames := map[string]interface{}{
			"SourceIP":             nil,
			"DestIP":               nil,
			"SourcePort":           nil,
			"SourceLabels":         nil,
			"DestServiceNamespace": nil,
			"DestServiceName":      nil,
			"DestServicePort":      nil,
			"DestLabels":           nil,
			"Policies":             nil,
			"OrigSourceIPs":        nil,
			"NumOrigSourceIPs":     nil,
		}
		// Use reflection to loop over the fields and ensure they all have non
		// zero values
		oType := reflect.TypeOf(out)
		oVal := reflect.ValueOf(out)
		for i := 0; i < oType.NumField(); i++ {
			field := oType.Field(i)
			zeroVal := reflect.Zero(field.Type)
			actualVal := oVal.Field(i)
			if _, ok := zeroFieldNames[field.Name]; ok {
				It(fmt.Sprintf("should not set %s", field.Name), func() {
					Expect(actualVal.Interface()).To(Equal(zeroVal.Interface()))
				})
			} else {
				It(fmt.Sprintf("should set %s", field.Name), func() {
					Expect(actualVal.Interface()).ToNot(Equal(zeroVal.Interface()))
				})
			}
		}
	})

	Describe("should not set source and destination ports for icmp flow", func() {
		flowLog := FlowLog{
			StartTime: time.Now(),
			EndTime:   time.Now(),
			FlowMeta: FlowMeta{
				Tuple: Tuple{
					proto: 1,
					src:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					dst:   [16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
					l4Src: 1234,
					l4Dst: 2948,
				},
				SrcMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "test",
					Name:           "test",
					AggregatedName: "test-*",
				},
				DstMeta: EndpointMetadata{
					Type:           "wep",
					Namespace:      "test",
					Name:           "test",
					AggregatedName: "test-*",
				},
				Action:   "allow",
				Reporter: "src",
			},
			FlowLabels: FlowLabels{
				SrcLabels: nil,
				DstLabels: nil,
			},
			FlowReportedStats: FlowReportedStats{
				PacketsIn:             1,
				PacketsOut:            2,
				BytesIn:               3,
				BytesOut:              4,
				NumFlowsStarted:       5,
				NumFlowsCompleted:     6,
				NumFlows:              7,
				HTTPRequestsAllowedIn: 8,
				HTTPRequestsDeniedIn:  9,
			},
			FlowExtras: FlowExtras{
				OriginalSourceIPs:    []net.IP{},
				NumOriginalSourceIPs: 0,
			},
		}

		out := toOutput(&flowLog)

		zeroFieldNames := map[string]interface{}{
			"SourceIP":             nil,
			"DestIP":               nil,
			"SourcePort":           nil,
			"DestPort":             nil,
			"DestServiceNamespace": nil,
			"DestServiceName":      nil,
			"DestServicePort":      nil,
			"SourceLabels":         nil,
			"DestLabels":           nil,
			"Policies":             nil,
			"OrigSourceIPs":        nil,
			"NumOrigSourceIPs":     nil,
		}
		// Use reflection to loop over the fields and ensure they all have non
		// zero values
		oType := reflect.TypeOf(out)
		oVal := reflect.ValueOf(out)
		for i := 0; i < oType.NumField(); i++ {
			field := oType.Field(i)
			zeroVal := reflect.Zero(field.Type)
			actualVal := oVal.Field(i)
			if _, ok := zeroFieldNames[field.Name]; ok {
				It(fmt.Sprintf("should not set %s", field.Name), func() {
					Expect(actualVal.Interface()).To(Equal(zeroVal.Interface()))
				})
			} else {
				It(fmt.Sprintf("should set %s", field.Name), func() {
					Expect(actualVal.Interface()).ToNot(Equal(zeroVal.Interface()))
				})
			}
		}
	})
})
