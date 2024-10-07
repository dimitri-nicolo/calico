package calicojson

import "encoding/json"

type Map map[string]interface{}

func MustMarshal(obj interface{}) []byte {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	return jsonBytes
}

func MustUnmarshalToStandObject(obj interface{}) interface{} {
	var jsonBytes []byte

	switch obj := obj.(type) {
	case []byte:
		jsonBytes = obj
	case string:
		jsonBytes = []byte(obj)
	default:
		jsonBytes = MustMarshal(obj)
	}

	stdObj := map[string]interface{}{}
	if err := json.Unmarshal(jsonBytes, &stdObj); err != nil {
		panic(err)
	}

	return stdObj
}
