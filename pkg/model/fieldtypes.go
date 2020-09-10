package model

import "fmt"

const (
	StringSliceFieldType = "stringSlice"
)

// Support function which converts a typed slice to []interface{}.
func convertToGenericSlice(fieldVal interface{}, fieldType string) ([]interface{}, error) {
	switch fieldType {
	case StringSliceFieldType:
		if strSlice, ok := fieldVal.([]string); ok {
			genericSlice := make([]interface{}, len(strSlice))
			for i, s := range strSlice {
				genericSlice[i] = s
			}
			return genericSlice, nil
		} else {
			return nil, fmt.Errorf("Error casting to string slice: %v", fieldVal)
		}
	default:
		return nil, fmt.Errorf("Unknown field type: %s", fieldType)
	}
}
