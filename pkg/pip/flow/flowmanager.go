package flow

import (
	"encoding/json"
)

// Returns a new FlowManager
func NewFlowManager() FlowManager {
	return &flowManager{}
}

// FlowManager implementation
type flowManager struct {
	data interface{}
}

// Marshal return the json encodeing of this FlowManager
func (m *flowManager) Marshal() ([]byte, error) {
	return json.Marshal(m.data)
}

// Unmarshal parses the json encoded data in to this FlowManager
func (m *flowManager) Unmarshal(b []byte) error {
	return json.Unmarshal(b, &m.data)
}

//tells us if the data has flows
func (m *flowManager) HasFlows() bool {
	agg, ok := m.data.(map[string]interface{})["aggregations"]
	if !ok {
		return false
	}
	_, ok = agg.(map[string]interface{})["flog_buckets"]
	if !ok {
		return false
	}
	return true
}

// Returns a slice of Flows extracted from the FlowManager
func (m *flowManager) ExtractFlows() ([]Flow, error) {

	//drill down to the flow log buckets
	fb := m.data.(map[string]interface{})["aggregations"].(map[string]interface{})["flog_buckets"]

	rb, err := json.Marshal(fb)
	if err != nil {
		return nil, err
	}

	return jsonToFlows(rb)
}

// Replaces the flows in the FlowManager with the provided flows
func (m *flowManager) ReplaceFlows(inflows []Flow) ([]byte, error) {

	//convert the flows back to json
	b, _ := flowsToJson(inflows)

	//unmarshal to an interface map
	var fl interface{}
	json.Unmarshal(b, &fl)

	//grab a handle to the flow log buckets
	flog_buckets := m.data.(map[string]interface{})["aggregations"].(map[string]interface{})["flog_buckets"]

	//update the flow log buckets with the flows
	flog_buckets.(map[string]interface{})["buckets"] = fl.(map[string]interface{})["buckets"]

	return json.Marshal(fl)
}

//Converts the json to a slice of Flows
func jsonToFlows(b []byte) ([]Flow, error) {

	flb := es_flog_buckets{}
	e := json.Unmarshal(b, &flb)
	if e != nil {
		return make([]Flow, 0), e
	}

	out := make([]Flow, len(flb.Flows))
	for i, fb := range flb.Flows {
		out[i] = fb.toFlow()
	}
	return out, nil
}

// Converts a slice of Flows to json byte slice
func flowsToJson(fls []Flow) ([]byte, error) {
	outflows := make([]es_flow, len(fls))
	for i, f := range fls {
		outflows[i] = es_flow{}
		outflows[i].fromFlow(f)
	}

	fb := es_flog_buckets{
		Flows: outflows,
	}
	return json.Marshal(fb)
}
