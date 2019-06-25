package flow

// A simplified Flow structure
type Flow struct {
	Reporter      string
	Src_type      string
	Src_NS        string
	Src_name      string
	Dest_type     string
	Dest_NS       string
	Dest_name     string
	Dest_port     string
	Dest_IP string
	Action        string
	Policies      []FlowPolicy
	Dest_labels   map[string]string
	Src_labels    map[string]string
	PreviewAction string
	Proto         string
}

type FlowPolicy struct {
	Order  int64
	Tier   string
	Name   string
	Action string
}

// FlowManager wraps the process of decoding json from elastic search
// into Flows and back the other way. The flow manager keeps the es
// json wrapper intact and reinserts the flows in the proper location.
// The primary use case is to allow flows to be modified in some way.
// The intended use case is:
// Unmarshal -> ExtractFlows -> ModifyFlows -> ReplaceFlows -> Marshal
type FlowManager interface {
	Unmarshal([]byte) error
	HasFlows() bool
	ExtractFlows() ([]Flow, error)
	ReplaceFlows([]Flow) ([]byte, error)
	Marshal() ([]byte, error)
}
