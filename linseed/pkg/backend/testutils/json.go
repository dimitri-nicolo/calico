// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package testutils

import (
	"strings"
	"testing"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"

	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/json"
)

func MustUnmarshalToMap(t *testing.T, source string) map[string]interface{} {
	var val map[string]interface{}
	err := json.Unmarshal([]byte(source), &val)
	require.NoError(t, err)
	return val
}

func MarshalBulkResponse(t *testing.T, response *v1.BulkResponse) string {
	newData, err := json.Marshal(response)
	require.NoError(t, err)

	return string(newData)
}

func MarshalBulkParams[T any](bulkParams []T) string {
	var logs []string

	for _, p := range bulkParams {
		newData, _ := json.Marshal(p)
		logs = append(logs, string(newData))
	}

	return strings.Join(logs, "\n")
}
