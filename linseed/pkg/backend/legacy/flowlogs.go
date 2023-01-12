package legacy

import (
	"context"
	"fmt"

	elastic "github.com/olivere/elastic/v7"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type FlowLogBackend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client
}

func NewFlowLogBackend(c lmaelastic.Client) *FlowLogBackend {
	return &FlowLogBackend{
		client:    c.Backend(),
		lmaclient: c,
	}
}

// TODO: This isn't a comprehensive list.
type FlowLog struct {
	DestType             string `json:"dest_type"`
	DestNamespace        string `json:"dest_namespace"`
	DestNameAggr         string `json:"dest_name_aggr"`
	DestServiceNamespace string `json:"dest_service_namespace"`
	DestServiceName      string `json:"dest_service_name"`
	DestServicePort      string `json:"dest_service_port"` // Deprecated
	DestServicePortNum   int    `json:"dest_service_port_num"`
	Protocol             string `json:"proto"`
	DestPort             int    `json:"dest_port"`
	SourceType           string `json:"source_type"`
	SourceNamespace      string `json:"source_namespace"`
	SourceNameAggr       string `json:"source_name_aggr"`
	ProcessName          string `json:"process_name"`
	Reporter             string `json:"reporter"`
	Action               string `json:"action"`

	PacketsIn               int `json:"packets_in"`
	PacketsOut              int `json:"packets_out"`
	BytesIn                 int `json:"bytes_in"`
	BytesOut                int `json:"bytes_out"`
	NumFlows                int `json:"num_flows"`
	NumFlowsStarted         int `json:"num_flows_started"`
	NumFlowsCompleted       int `json:"num_flows_completed"`
	TCPTotalRetransmissions int `json:"tcp_total_retransmissions"`
	TCPLostPackets          int `json:"tcp_lost_packets"`
	TCPUnrecoveredTo        int `json:"tcp_unrecovered_to"`

	NumProcessNames             int `json:"num_process_names"`
	NumProcessIDs               int `json:"num_process_i_ds"`
	TCPMinSendCongestionWindow  int `json:"tcp_min_send_congestion_window"`
	TCPMinMSS                   int `json:"tcp_min_mss"`
	TCPMaxSmoothRTT             int `json:"tcp_max_smooth_rtt"`
	TCPMaxMinRTT                int `json:"tcp_max_min_rtt"`
	TCPMeanSendCongestionWindow int `json:"tcp_mean_send_congestion_window"`
	TCPMeanMSS                  int `json:"tcp_mean_mss"`

	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`

	// Cluster should not be set directly by calling code.
	// Rather, this is set by the backend when creating the flow log.
	Cluster string `json:"cluster,omitempty"`
}

func (b *FlowLogBackend) Initialize(ctx context.Context) error {
	// Create a template with mappings for all new flow log indices.
	_, err := b.client.IndexPutTemplate("flow_log_template").
		BodyString(templates.FlowLogTemplate).
		Create(false).
		Do(ctx)
	return err
}

// Create the given flow log in elasticsearch.
func (b *FlowLogBackend) Create(ctx context.Context, i bapi.ClusterInfo, f FlowLog) error {
	// TODO: Don't re-use the GET logger.
	log := contextLogger(v1.L3FlowParams{})

	if i.Cluster == "" {
		log.Fatal("BUG: No cluster ID on request")
	}
	if f.Cluster != "" {
		// For the legacy backend, the Cluster ID is encoded into the index
		// and not the log itself.
		log.Fatal("BUG: Cluster ID should not be set on flow log")
	}

	// TODO: Move this elsewhere.
	err := b.Initialize(ctx)
	if err != nil {
		return err
	}

	// Determine the index to write to. It will be automatically created based on the configured
	// flow template if it does not already exist.
	index := fmt.Sprintf("tigera_secure_ee_flows.%s", i.Cluster)
	log.Infof("Creating flow log in index %s", index)

	// Add the flow log to the index.
	// TODO: Probably want this to be the /bulk endpoint.
	resp, err := b.client.Index().
		Index(index).
		BodyJson(f).
		Refresh("true"). // TODO: Probably want this to be false.
		Do(ctx)
	if err != nil {
		log.Errorf("Error writing flow log: %s", err)
		return fmt.Errorf("failed to write flow log: %s", err)
	}

	log.Infof("Wrote flow log to index: %+v", resp)

	return nil
}
