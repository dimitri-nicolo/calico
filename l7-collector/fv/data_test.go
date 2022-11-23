// Copyright (c) 2020 Tigera, Inc. All rights reserved.

package fv_test

import (
	"github.com/projectcalico/calico/l7-collector/proto"
)

var (
	httpLog  = "{\"downstream_remote_address\":\"192.168.138.208:34368\",\"downstream_local_address\":\"192.168.35.210:80\",\"type\":\"HTTP/1.1\",\"duration\":3,\"user_agent\":\"curl/7.68.0\",\"response_code\":200,\"request_id\":\"e23c0019-36b7-4142-8e86-39d15b00e965\",\"bytes_received\":1,\"request_path\":\"/ip\",\"reporter\":\"destination\",\"bytes_sent\":33,\"request_method\":\"GET\",\"domain\":\"http-service\"}\n"
	httpLog2 = "{\"downstream_remote_address\":\"192.168.138.208:34368\",\"downstream_local_address\":\"192.168.35.210:80\",\"type\":\"HTTP/1.1\",\"duration\":3,\"user_agent\":\"curl/7.68.0\",\"response_code\":200,\"request_id\":\"36b7-4142-8e86-39d15b00e965-e23c0019\",\"bytes_received\":1,\"request_path\":\"/ip\",\"reporter\":\"destination\",\"bytes_sent\":33,\"request_method\":\"GET\",\"domain\":\"http-service\"}\n"
	httpLog3 = "{\"downstream_remote_address\":\"192.168.138.208:34368\",\"downstream_local_address\":\"192.168.35.210:80\",\"type\":\"HTTP/1.1\",\"duration\":3,\"user_agent\":\"curl/7.68.0\",\"response_code\":200,\"request_id\":\"4142-8e86-39d15b00e965-e23c0019-36b7\",\"bytes_received\":1,\"request_path\":\"/ip\",\"reporter\":\"destination\",\"bytes_sent\":33,\"request_method\":\"GET\",\"domain\":\"http-service\"}\n"
	httpLog4 = "{\"downstream_remote_address\":\"192.168.138.208:34368\",\"downstream_local_address\":\"192.168.35.210:80\",\"type\":\"HTTP/1.1\",\"duration\":3,\"user_agent\":\"curl/7.68.0\",\"response_code\":200,\"request_id\":\"8e86-39d15b00e965-e23c0019-36b7-4142\",\"bytes_received\":0,\"request_path\":\"/ip\",\"reporter\":\"destination\",\"bytes_sent\":33,\"request_method\":\"GET\",\"domain\":\"http-service\"}\n"
	httpLog5 = "{\"downstream_remote_address\":\"192.168.138.208:34368\",\"downstream_local_address\":\"192.168.35.210:80\",\"type\":\"HTTP/1.1\",\"duration\":3,\"user_agent\":\"curl/7.68.0\",\"response_code\":200,\"request_id\":\"236b7-4142-8e86-39d15-b00e-96536bcc7\",\"bytes_received\":0,\"request_path\":\"/ip\",\"reporter\":\"destination\",\"bytes_sent\":33,\"request_method\":\"GET\",\"domain\":\"http-service\"}\n"
	httpLog6 = "{\"downstream_remote_address\":\"192.168.138.208:34368\",\"downstream_local_address\":\"192.168.35.210:80\",\"type\":\"HTTP/1.1\",\"duration\":3,\"user_agent\":\"curl/7.68.0\",\"response_code\":200,\"request_id\":\"96536b7-236b7-4142-8e86-39d15-b00cce\",\"bytes_received\":0,\"request_path\":\"/ip\",\"reporter\":\"destination\",\"bytes_sent\":33,\"request_method\":\"GET\",\"domain\":\"http-service\"}\n"
	httpLog7 = "{\"downstream_remote_address\":\"192.168.138.208:34368\",\"downstream_local_address\":\"192.168.35.210:80\",\"type\":\"HTTP/1.1\",\"duration\":3,\"user_agent\":\"curl/7.68.0\",\"response_code\":200,\"request_id\":\"96536b7-4142-8e86-39d15-b00cce236b7d\",\"bytes_received\":0,\"request_path\":\"/ip\",\"reporter\":\"destination\",\"bytes_sent\":33,\"request_method\":\"GET\",\"domain\":\"http-service\"}\n"
	tcpLog   = "{\"duration\":2,\"downstream_local_address\":\"192.168.138.227:6379\",\"response_code\":0,\"user_agent\":null,\"start_time\":\"2020-12-08T23:05:04.119Z\",\"request_id\":null,\"bytes_received\":14,\"request_path\":null,\"type\":\"tcp\",\"reporter\":\"destination\",\"bytes_sent\":7, \"request_method\":null,\"downstream_remote_address\":\"192.168.7.147:38600\"}\n"
	tcpLog2  = "{\"duration\":4,\"downstream_local_address\":\"192.168.138.227:6379\",\"response_code\":0,\"user_agent\":null,\"start_time\":\"2020-12-08T23:05:04.119Z\",\"request_id\":null,\"bytes_received\":4, \"request_path\":null,\"type\":\"tcp\",\"reporter\":\"destination\",\"bytes_sent\":3, \"request_method\":null,\"downstream_remote_address\":\"192.168.7.147:38600\"}\n"
	tcpLog3  = "{\"duration\":8,\"downstream_local_address\":\"192.168.138.227:6379\",\"response_code\":0,\"user_agent\":null,\"start_time\":\"2020-12-08T23:05:04.119Z\",\"request_id\":null,\"bytes_received\":4, \"request_path\":null,\"type\":\"tcp\",\"reporter\":\"destination\",\"bytes_sent\":10,\"request_method\":null,\"downstream_remote_address\":\"192.168.7.147:38600\"}\n"

	httpPostLog   = "{\"downstream_remote_address\":\"192.168.138.208:34368\",\"downstream_local_address\":\"192.168.35.210:80\",\"type\":\"HTTP/1.1\",\"duration\":3,\"user_agent\":\"curl/7.68.0\",\"response_code\":200,\"request_id\":\"96536b7-4142-8e86-39d15-b00cce236b7d\",\"bytes_received\":1,\"request_path\":\"/ip\",\"reporter\":\"destination\",\"bytes_sent\":33,\"request_method\":\"POST\",\"domain\":\"http-service\"}\n"
	httpDeleteLog = "{\"downstream_remote_address\":\"192.168.138.208:34368\",\"downstream_local_address\":\"192.168.35.210:80\",\"type\":\"HTTP/1.1\",\"duration\":3,\"user_agent\":\"curl/7.68.0\",\"response_code\":200,\"request_id\":\"96536b7-4142-8e86-39d15-b00cce236b7d\",\"bytes_received\":1,\"request_path\":\"/ip\",\"reporter\":\"destination\",\"bytes_sent\":33,\"request_method\":\"DELETE\",\"domain\":\"http-service\"}\n"
	sourceLog     = "{\"downstream_remote_address\":\"192.168.138.208:34368\",\"downstream_local_address\":\"192.168.35.210:80\",\"type\":\"HTTP/1.1\",\"duration\":3,\"user_agent\":\"curl/7.68.0\",\"response_code\":200,\"request_id\":\"96536b7-4142-8e86-39d15-b00cce236b7d\",\"bytes_received\":1,\"request_path\":\"/ip\",\"reporter\":\"source\",\"bytes_sent\":33,\"request_method\":\"GET\",\"domain\":\"http-service\"}\n"

	httpStat = &proto.DataplaneStats{
		SrcIp:    "192.168.138.208",
		DstIp:    "192.168.35.210",
		SrcPort:  int32(34368),
		DstPort:  int32(80),
		Protocol: &proto.Protocol{NumberOrName: &proto.Protocol_Name{Name: "tcp"}},
		HttpData: []*proto.HttpData{
			&proto.HttpData{
				Duration:      3,
				ResponseCode:  200,
				BytesSent:     33,
				BytesReceived: 1,
				UserAgent:     "curl/7.68.0",
				RequestPath:   "/ip",
				RequestMethod: "GET",
				Type:          "HTTP/1.1",
				Count:         1,
				Domain:        "http-service",
				DurationMax:   3,
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

	httpStatSummation = &proto.DataplaneStats{
		SrcIp:    "192.168.138.208",
		DstIp:    "192.168.35.210",
		SrcPort:  int32(34368),
		DstPort:  int32(80),
		Protocol: &proto.Protocol{NumberOrName: &proto.Protocol_Name{Name: "tcp"}},
		HttpData: []*proto.HttpData{
			&proto.HttpData{
				Duration:      9,
				ResponseCode:  200,
				BytesSent:     99,
				BytesReceived: 3,
				UserAgent:     "curl/7.68.0",
				RequestPath:   "/ip",
				RequestMethod: "GET",
				Type:          "HTTP/1.1",
				Count:         3,
				Domain:        "http-service",
				DurationMax:   3,
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

	httpBatchStat3 = &proto.DataplaneStats{
		SrcIp:    "192.168.138.208",
		DstIp:    "192.168.35.210",
		SrcPort:  int32(34368),
		DstPort:  int32(80),
		Protocol: &proto.Protocol{NumberOrName: &proto.Protocol_Name{Name: "tcp"}},
		HttpData: []*proto.HttpData{
			{
				Duration:      3,
				ResponseCode:  200,
				BytesSent:     33,
				BytesReceived: 1,
				UserAgent:     "curl/7.68.0",
				RequestPath:   "/ip",
				RequestMethod: "GET",
				Type:          "HTTP/1.1",
				Count:         1,
				Domain:        "http-service",
				DurationMax:   3,
			},
			{
				Duration:      3,
				ResponseCode:  200,
				BytesSent:     33,
				BytesReceived: 1,
				UserAgent:     "curl/7.68.0",
				RequestPath:   "/ip",
				RequestMethod: "POST",
				Type:          "HTTP/1.1",
				Count:         1,
				Domain:        "http-service",
				DurationMax:   3,
			},
			{
				Duration:      3,
				ResponseCode:  200,
				BytesSent:     33,
				BytesReceived: 1,
				UserAgent:     "curl/7.68.0",
				RequestPath:   "/ip",
				RequestMethod: "DELETE",
				Type:          "HTTP/1.1",
				Count:         1,
				Domain:        "http-service",
				DurationMax:   3,
			},
		},
		Stats: []*proto.Statistic{
			{
				Direction:  proto.Statistic_IN,
				Relativity: proto.Statistic_DELTA,
				Kind:       proto.Statistic_HTTP_DATA,
				Action:     proto.Action_ALLOWED,
				Value:      int64(3),
			},
		},
	}
)
