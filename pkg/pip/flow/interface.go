package flow

// A simplified Flow structure
type Flow struct {
	Source        FlowEndpointData
	Dest          FlowEndpointData
	Reporter      string
	Action        string
	PreviewAction string
	Policies      []FlowPolicy
	Proto         string
}

// FlowEndpointData can be used to describe the source or destination
// of a flow log.
type FlowEndpointData struct {
	Type      string
	Namespace string
	IP        string
	Name      string
	Labels    map[string]string
	Port      string
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
