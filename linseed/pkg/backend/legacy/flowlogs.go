package legacy

import (
	"context"
	"fmt"
	"net"

	elastic "github.com/olivere/elastic/v7"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
	"github.com/projectcalico/calico/linseed/pkg/backend/legacy/templates"
	lmaelastic "github.com/projectcalico/calico/lma/pkg/elastic"
)

type FlowLogBackend struct {
	client    *elastic.Client
	lmaclient lmaelastic.Client

	// Tracks whether the backend has been initialized.
	initialized bool
}

func NewFlowLogBackend(c lmaelastic.Client) *FlowLogBackend {
	b := &FlowLogBackend{
		client:    c.Backend(),
		lmaclient: c,
	}
	return b
}

type FlowLogPolicy struct {
	AllPolicies string `json:"all_policies"`
}

type FlowLogLabels struct {
	Labels string `json:"labels"`
}

type FlowLog struct {
	// Destination fields.
	DestType             string        `json:"dest_type"`
	DestIP               net.IP        `json:"dest_ip"`
	DestNamespace        string        `json:"dest_namespace"`
	DestNameAggr         string        `json:"dest_name_aggr"`
	DestPort             int           `json:"dest_port"`
	DestLabels           FlowLogLabels `json:"dest_labels"`
	DestServiceNamespace string        `json:"dest_service_namespace"`
	DestServiceName      string        `json:"dest_service_name"`
	DestServicePort      string        `json:"dest_service_port"` // Deprecated
	DestServicePortNum   int           `json:"dest_service_port_num"`
	DestDomains          string        `json:"dest_domains"`

	// Source fields.
	SourceType           string        `json:"source_type"`
	SourceIP             net.IP        `json:"source_ip"`
	SourceNamespace      string        `json:"source_namespace"`
	SourceNameAggr       string        `json:"source_name_aggr"`
	SourcePort           int           `json:"source_port"`
	SourceLabels         FlowLogLabels `json:"source_labels"`
	OriginalSourceIPs    net.IP        `json:"original_source_i_ps"`
	NumOriginalSourceIPs int           `json:"num_original_source_ips"`

	// Reporter is src or dest - the location where this flowlog was generated.
	Reporter         string          `json:"reporter"`
	Protocol         string          `json:"proto"`
	Action           string          `json:"action"`
	NATOutgoingPorts int             `json:"nat_outgoing_ports"`
	Policies         []FlowLogPolicy `json:"policies"`

	// HTTP fields.
	HTTPRequestsAllowedIn int `json:"http_requests_allowed_in"`
	HTTPRequestsDeniedIn  int `json:"http_requests_denied_in"`

	// Traffic stats.
	PacketsIn  int `json:"packets_in"`
	PacketsOut int `json:"packets_out"`
	BytesIn    int `json:"bytes_in"`
	BytesOut   int `json:"bytes_out"`

	// Stats from the original logs used to generate this flow log.
	// Felix aggregates multiple flow logs into a single FlowLog.
	NumFlows          int `json:"num_flows"`
	NumFlowsStarted   int `json:"num_flows_started"`
	NumFlowsCompleted int `json:"num_flows_completed"`

	// Process stats.
	NumProcessNames int    `json:"num_process_names"`
	NumProcessIDs   int    `json:"num_process_ids"`
	ProcessName     string `json:"process_name"`
	NumProcessArgs  int    `json:"num_process_args"`
	ProcessArgs     string `json:"process_args"`

	// TCP stats.
	TCPMinSendCongestionWindow  int `json:"tcp_min_send_congestion_window"`
	TCPMinMSS                   int `json:"tcp_min_mss"`
	TCPMaxSmoothRTT             int `json:"tcp_max_smooth_rtt"`
	TCPMaxMinRTT                int `json:"tcp_max_min_rtt"`
	TCPMeanSendCongestionWindow int `json:"tcp_mean_send_congestion_window"`
	TCPMeanMSS                  int `json:"tcp_mean_mss"`
	TCPMeanMinRTT               int `json:"tcp_mean_min_rtt"`
	TCPMeanSmoothRTT            int `json:"tcp_mean_smooth_rtt"`
	TCPTotalRetransmissions     int `json:"tcp_total_retransmissions"`
	TCPLostPackets              int `json:"tcp_lost_packets"`
	TCPUnrecoveredTo            int `json:"tcp_unrecovered_to"`

	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`

	// Cluster should not be set directly by calling code.
	// Rather, this is set by the backend when creating the flow log.
	Cluster string `json:"cluster,omitempty"`
}

func (b *FlowLogBackend) Initialize(ctx context.Context) error {
	var err error
	if !b.initialized {
		// Create a template with mappings for all new flow log indices.
		_, err = b.client.IndexPutTemplate("flow_log_template").
			BodyString(templates.FlowLogTemplate).
			Create(false).
			Do(ctx)
		if err != nil {
			return err
		}
		b.initialized = true
	}
	return nil
}

// Create the given flow log in elasticsearch.
func (b *FlowLogBackend) Create(ctx context.Context, i bapi.ClusterInfo, f FlowLog) error {
	log := contextLogger(i)

	// Initialize if we haven't yet.
	err := b.Initialize(ctx)
	if err != nil {
		return err
	}

	if i.Cluster == "" {
		log.Fatal("BUG: No cluster ID on request")
	}
	if f.Cluster != "" {
		// For the legacy backend, the Cluster ID is encoded into the index
		// and not the log itself.
		log.Fatal("BUG: Cluster ID should not be set on flow log")
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
