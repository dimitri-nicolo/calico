// Copyright (c) 2019 Tigera, Inc. All rights reserved.

package fv_test

import (
	"github.com/tigera/ingress-collector/proto"
)

var (
	basicLog       = "tigera_secure_ee_ingress: {\"source_port\": 1, \"destination_ip\": \"10.1.1.10\", \"destination_port\": 2, \"source_ip\": \"10.1.10.1\", \"x-forwarded-for\": \"1.1.1.1\", \"x-real-ip\": \"2.2.2.2\"}\n"
	basicLog2      = "tigera_secure_ee_ingress: {\"source_port\": 1, \"destination_ip\": \"10.1.1.10\", \"destination_port\": 2, \"source_ip\": \"10.1.10.1\", \"x-forwarded-for\": \"8.8.8.8\", \"x-real-ip\": \"3.3.3.3\"}\n"
	basicLog3      = "tigera_secure_ee_ingress: {\"source_port\": 1, \"destination_ip\": \"10.1.1.10\", \"destination_port\": 2, \"source_ip\": \"10.1.10.1\", \"x-forwarded-for\": \"9.9.9.9\", \"x-real-ip\": \"4.4.4.4\"}\n"
	basicLogRepeat = "tigera_secure_ee_ingress: {\"source_port\": 1, \"destination_ip\": \"10.1.1.10\", \"destination_port\": 2, \"source_ip\": \"10.1.10.1\", \"x-forwarded-for\": \"1.1.1.1\", \"x-real-ip\": \"2.2.2.2\"}\n"
	basicLog4      = "tigera_secure_ee_ingress: {\"source_port\": 1, \"destination_ip\": \"10.1.1.10\", \"destination_port\": 2, \"source_ip\": \"10.1.10.1\", \"x-forwarded-for\": \"10.10.10.10\", \"x-real-ip\": \"5.5.5.5\"}\n"
	basicLog5      = "tigera_secure_ee_ingress: {\"source_port\": 1, \"destination_ip\": \"10.1.1.10\", \"destination_port\": 2, \"source_ip\": \"10.1.10.1\", \"x-forwarded-for\": \"11.11.11.11\", \"x-real-ip\": \"6.6.6.6\"}\n"
	basicLog6      = "tigera_secure_ee_ingress: {\"source_port\": 1, \"destination_ip\": \"10.1.1.10\", \"destination_port\": 2, \"source_ip\": \"10.1.10.1\", \"x-forwarded-for\": \"12.12.12.12\", \"x-real-ip\": \"7.7.7.7\"}\n"
	badLog         = "tsee: {\"source_port\": 1, \"destination_ip\": \"10.1.1.10\", \"destination_port\": 2, \"source_ip\": \"10.1.10.1\", \"x-forwarded-for\": \"12.12.12.12\", \"x-real-ip\": \"7.7.7.7\"}\n"
	basicLogConn   = "tigera_secure_ee_ingress: {\"source_port\": 10, \"destination_ip\": \"10.1.1.10\", \"destination_port\": 20, \"source_ip\": \"10.1.10.1\", \"x-forwarded-for\": \"13.13.13.13\", \"x-real-ip\": \"10.10.10.10\"}\n"
	basicLogDps    = &proto.DataplaneStats{
		SrcIp:    "10.1.10.1",
		DstIp:    "10.1.1.10",
		SrcPort:  int32(1),
		DstPort:  int32(2),
		Protocol: &proto.Protocol{&proto.Protocol_Name{"tcp"}},
		HttpData: []*proto.HttpData{
			&proto.HttpData{
				XForwardedFor: "1.1.1.1",
				XRealIp:       "2.2.2.2",
			},
		},
		Stats: []*proto.Statistic{
			&proto.Statistic{
				Direction:  proto.Statistic_IN,
				Relativity: proto.Statistic_DELTA,
				Kind:       proto.Statistic_HTTP_DATA,
				Action:     proto.Action_ALLOWED,
				Value:      int64(1),
			},
		},
	}
	basicLogDpsMultiple = &proto.DataplaneStats{
		SrcIp:    "10.1.10.1",
		DstIp:    "10.1.1.10",
		SrcPort:  int32(1),
		DstPort:  int32(2),
		Protocol: &proto.Protocol{&proto.Protocol_Name{"tcp"}},
		HttpData: []*proto.HttpData{
			&proto.HttpData{
				XForwardedFor: "1.1.1.1",
				XRealIp:       "2.2.2.2",
			},
			&proto.HttpData{
				XForwardedFor: "8.8.8.8",
				XRealIp:       "3.3.3.3",
			},
			&proto.HttpData{
				XForwardedFor: "9.9.9.9",
				XRealIp:       "4.4.4.4",
			},
		},
		Stats: []*proto.Statistic{
			&proto.Statistic{
				Direction:  proto.Statistic_IN,
				Relativity: proto.Statistic_DELTA,
				Kind:       proto.Statistic_HTTP_DATA,
				Action:     proto.Action_ALLOWED,
				Value:      int64(3),
			},
		},
	}
	basicLogDpsLimit = &proto.DataplaneStats{
		SrcIp:    "10.1.10.1",
		DstIp:    "10.1.1.10",
		SrcPort:  int32(1),
		DstPort:  int32(2),
		Protocol: &proto.Protocol{&proto.Protocol_Name{"tcp"}},
		HttpData: []*proto.HttpData{
			&proto.HttpData{
				XForwardedFor: "1.1.1.1",
				XRealIp:       "2.2.2.2",
			},
			&proto.HttpData{
				XForwardedFor: "8.8.8.8",
				XRealIp:       "3.3.3.3",
			},
			&proto.HttpData{
				XForwardedFor: "9.9.9.9",
				XRealIp:       "4.4.4.4",
			},
			&proto.HttpData{
				XForwardedFor: "10.10.10.10",
				XRealIp:       "5.5.5.5",
			},
			&proto.HttpData{
				XForwardedFor: "11.11.11.11",
				XRealIp:       "6.6.6.6",
			},
		},
		Stats: []*proto.Statistic{
			&proto.Statistic{
				Direction:  proto.Statistic_IN,
				Relativity: proto.Statistic_DELTA,
				Kind:       proto.Statistic_HTTP_DATA,
				Action:     proto.Action_ALLOWED,
				Value:      int64(6),
			},
		},
	}
	basicLogDpsConn = &proto.DataplaneStats{
		SrcIp:    "10.1.10.1",
		DstIp:    "10.1.1.10",
		SrcPort:  int32(10),
		DstPort:  int32(20),
		Protocol: &proto.Protocol{&proto.Protocol_Name{"tcp"}},
		HttpData: []*proto.HttpData{
			&proto.HttpData{
				XForwardedFor: "13.13.13.13",
				XRealIp:       "10.10.10.10",
			},
		},
		Stats: []*proto.Statistic{
			&proto.Statistic{
				Direction:  proto.Statistic_IN,
				Relativity: proto.Statistic_DELTA,
				Kind:       proto.Statistic_HTTP_DATA,
				Action:     proto.Action_ALLOWED,
				Value:      int64(1),
			},
		},
	}
)
