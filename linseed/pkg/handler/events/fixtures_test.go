package events

import (
	"time"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
)

const dummyURL = "anyURL"

var (
	noEvents []v1.Event

	srcIP         = "74.125.124.100"
	srcPort       = int64(9090)
	destIP        = "10.28.0.13"
	NodeName      = "node0"
	destName      = "event-destination"
	destNamespace = "event-dest-ns"

	event = v1.Event{
		Time:          v1.NewEventTimestamp(time.Now().Unix()),
		Type:          "deep_packet_inspection",
		Description:   "Encountered suspicious traffic matching snort rule for malicious activity",
		Severity:      100,
		Origin:        "dpi.dpi-ns/dpi-name",
		SourceIP:      &srcIP,
		SourcePort:    &srcPort,
		DestIP:        &destIP,
		DestName:      destName,
		DestNamespace: destNamespace,
		Host:          NodeName,
		Record:        v1.DPIRecord{SnortSignatureID: "1000005", SnortSignatureRevision: "1", SnortAlert: "21/08/30-17:19:37.337831 [**] [1:1000005:1] \"msg:1_alert_fast\" [**] [Priority: 0] {ICMP} 74.125.124.100:9090 -> 10.28.0.13"},
	}

	multipleEvent = []v1.Event{
		event, event,
	}
)

var bulkResponseSuccess = &v1.BulkResponse{
	Total:     2,
	Succeeded: 2,
	Failed:    0,
}

var bulkResponsePartialSuccess = &v1.BulkResponse{
	Total:     2,
	Succeeded: 1,
	Failed:    1,
	Errors: []v1.BulkError{
		{
			Resource: "res",
			Type:     "index error",
			Reason:   "I couldn't do it",
		},
	},
}
