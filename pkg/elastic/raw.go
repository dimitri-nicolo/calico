package elastic

import "encoding/json"

// RawElasticQuery is a wrapper around a raw JSON message so that it can implement the elastic Query interface.
type RawElasticQuery json.RawMessage

func (r RawElasticQuery) Source() (interface{}, error) {
	return json.RawMessage(r), nil
}
