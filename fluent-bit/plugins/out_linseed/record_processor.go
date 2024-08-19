// Copyright (c) 2024 Tigera, Inc. All rights reserved.
package main

import (
	"bytes"
	"encoding/json"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
)

type Record map[interface{}]interface{}

type RecordProcessor struct{}

func NewRecordProcessor() *RecordProcessor {
	return &RecordProcessor{}
}

func (rp *RecordProcessor) Process(data unsafe.Pointer, length int) (*bytes.Buffer, int, error) {
	var ndjsonBuffer bytes.Buffer

	// decode fluent-bit internal msgpack buffer to ndjson
	dec := output.NewDecoder(data, length)
	count := 0
	for {
		rc, _, record := output.GetRecord(dec)
		if rc != 0 {
			break
		}

		jsonData, err := json.Marshal(rp.toStringMap(record))
		if err != nil {
			return nil, count, err
		}

		ndjsonBuffer.Write(jsonData)
		ndjsonBuffer.WriteByte('\n')
		count++
	}

	return &ndjsonBuffer, count, nil
}

// prevent base64-encoding []byte values (default json.Encoder rule) by
// converting them to strings
func (rp *RecordProcessor) toStringSlice(slice []interface{}) []interface{} {
	var s []interface{}
	for _, v := range slice {
		switch t := v.(type) {
		case []byte:
			s = append(s, string(t))
		case map[interface{}]interface{}:
			s = append(s, rp.toStringMap(t))
		case []interface{}:
			s = append(s, rp.toStringSlice(t))
		default:
			s = append(s, t)
		}
	}
	return s
}

func (rp *RecordProcessor) toStringMap(record Record) map[string]interface{} {
	m := make(map[string]interface{})
	for k, v := range record {
		key, ok := k.(string)
		if !ok {
			continue
		}
		switch t := v.(type) {
		case []byte:
			m[key] = string(t)
		case map[interface{}]interface{}:
			m[key] = rp.toStringMap(t)
		case []interface{}:
			m[key] = rp.toStringSlice(t)
		default:
			m[key] = v
		}
	}
	return m
}
