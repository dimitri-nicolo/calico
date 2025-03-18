// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package testutils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/json"
)

func MustUnmarshalToMap(t *testing.T, source string) map[string]interface{} {
	var val map[string]interface{}
	err := json.Unmarshal([]byte(source), &val)
	require.NoError(t, err)
	return val
}

func Marshal(t *testing.T, response interface{}) string {
	newData, err := json.Marshal(response)
	require.NoError(t, err)

	return string(newData)
}

func MarshalBulkParams[T any](bulkParams []T) string {
	var logs []string

	for _, p := range bulkParams {
		newData, err := json.Marshal(p)
		if err != nil {
			panic(err)
		}
		logs = append(logs, string(newData))
	}

	return strings.Join(logs, "\n")
}
