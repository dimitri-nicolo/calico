// Copyright 2021 Tigera Inc. All rights reserved.

package maputil

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Copy returns a copy of the given map.
func Copy(src map[string]interface{}) (map[string]interface{}, error) {
	jsonString, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	dst := make(map[string]interface{})
	err = json.Unmarshal(jsonString, &dst)
	if err != nil {
		return nil, err
	}
	return dst, nil
}

// CreateLabelValuePairStr returns a string of the combined key value pairs of the map
// in format key0=value0,key1=Value1 suited for a resource's label
func CreateLabelValuePairStr(labelMap map[string]string) string {
	if labelMap == nil {
		return ""
	}

	var labels []string
	for key, value := range labelMap {
		labelKeyValueStr := fmt.Sprintf("%s=%s", key, value)
		labels = append(labels, labelKeyValueStr)
	}

	return strings.Join(labels, ",")
}
