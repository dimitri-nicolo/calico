// Copyright 2021 Tigera Inc. All rights reserved.

package maputil

import "encoding/json"

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
