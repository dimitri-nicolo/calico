// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package testutils

import (
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
