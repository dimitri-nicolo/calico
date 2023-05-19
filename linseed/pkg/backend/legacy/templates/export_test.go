package templates

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func AssertStructAndMap(t *testing.T, logType interface{}, mappings map[string]interface{}) bool {
	require.NotNil(t, mappings)
	if len(mappings) != 2 {
		return false
	}
	// Check Dynamic is false
	require.NotNil(t, mappings["dynamic"])
	require.Equal(t, "false", mappings["dynamic"])

	//Fetch Properties from the json template
	require.NotNil(t, mappings["properties"])
	properties := mappings["properties"].(map[string]interface{})

	obj := reflect.ValueOf(logType).Type()
	if obj.NumField() != len(properties) {
		return false
	}

	// Check each field in the struct exist in json template mapping
	for i := 0; i < obj.NumField(); i++ {
		field := obj.Field(i)
		tags := strings.Split(field.Tag.Get("json"), ",")
		val := ""
		if len(tags) > 0 {
			val = tags[0]
		}
		require.NotEmpty(t, val)
		if _, ok := properties[val]; !ok {
			return false
		}
	}
	return true
}
