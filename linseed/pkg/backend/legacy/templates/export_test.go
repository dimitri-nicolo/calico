package templates

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func AssertStructAndMap(t *testing.T, logType interface{}, mappings map[string]interface{}, matchCount bool, shouldSucceed bool) {

	require.Equal(t, 2, len(mappings))

	// Check Dynamic is false
	require.Equal(t, "false", mappings["dynamic"])

	//Fetch Properties from the json template
	var properties map[string]interface{}
	properties = mappings["properties"].(map[string]interface{})

	obj := reflect.ValueOf(logType).Type()

	if matchCount {
		require.Equal(t, obj.NumField(), len(properties))

		fieldExist := true
		// Check each field in the struct exist in json template mapping
		for i := 0; i < obj.NumField(); i++ {
			field := obj.Field(i)
			val := strings.Split(field.Tag.Get("json"), ",")[0]
			require.NotNil(t, val)
			fieldExist = fieldExist && CheckMapForField(val, properties)

		}
		require.Equal(t, shouldSucceed, fieldExist)
	} else {
		require.NotEqual(t, obj.NumField(), len(properties))
	}
}

// function to check if an expected field is present in the map
func CheckMapForField(expected string, mappings map[string]interface{}) bool {
	for fieldName, _ := range mappings {
		if fieldName == expected {
			return true
		}
	}
	return false
}
