// Copyright (c) 2023 Tigera, Inc. All rights reserved.
//

package testutils

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/stretchr/testify/require"

	"github.com/projectcalico/calico/libcalico-go/lib/json"
	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	bapi "github.com/projectcalico/calico/linseed/pkg/backend/api"
)

func AssertLogIDAndCopyFlowLogsWithoutID(t *testing.T, r *v1.List[v1.FlowLog]) []v1.FlowLog {
	require.NotNil(t, r)

	// Asert that we have an ID assigned from Elastic
	var copyOfLogs []v1.FlowLog
	for _, item := range r.Items {
		item = AssertFlowLogIDAndReset(t, item)
		copyOfLogs = append(copyOfLogs, item)
	}
	return copyOfLogs
}

func AssertFlowLogIDAndReset(t *testing.T, item v1.FlowLog) v1.FlowLog {
	require.NotEmpty(t, item.ID)
	item.ID = ""

	require.NotNil(t, item.GeneratedTime)
	item.GeneratedTime = nil

	return item
}

func AssertEventIDAndReset(t *testing.T, item v1.Event) v1.Event {
	require.NotEmpty(t, item.ID)
	item.ID = ""

	return item
}

func AssertLogIDAndCopyDNSLogsWithoutID(t *testing.T, r *v1.List[v1.DNSLog]) []v1.DNSLog {
	require.NotNil(t, r)

	// Asert that we have an ID assigned from Elastic
	var copyOfLogs []v1.DNSLog
	for _, item := range r.Items {
		item = AssertDNSLogIDAndReset(t, item)
		copyOfLogs = append(copyOfLogs, item)
	}
	return copyOfLogs
}

func AssertDNSLogIDAndReset(t *testing.T, item v1.DNSLog) v1.DNSLog {
	require.NotEmpty(t, item.ID)
	item.ID = ""

	// Similarly for GeneratedTime field, as test code cannot predict the exact value that
	// Linseed will populate here.
	require.NotNil(t, item.GeneratedTime)
	item.GeneratedTime = nil

	return item
}

func AssertLogIDAndCopyEventsWithoutID(t *testing.T, r *v1.List[v1.Event]) []v1.Event {
	require.NotNil(t, r)

	// Asert that we have an ID assigned from Elastic
	var copyOfEvents []v1.Event
	for _, item := range r.Items {
		item = AssertEventIDAndReset(t, item)
		copyOfEvents = append(copyOfEvents, item)
	}
	return copyOfEvents
}

func AssertRuntimeReportIDAndReset(t *testing.T, item v1.RuntimeReport) v1.RuntimeReport {
	require.NotEmpty(t, item.ID)
	item.ID = ""

	return item
}

func AssertLogIDAndCopyRuntimeReportsWithoutThem(t *testing.T, r *v1.List[v1.RuntimeReport]) []v1.RuntimeReport {
	require.NotNil(t, r)

	// Asert that we have an ID assigned from Elastic
	var copyOfReports []v1.RuntimeReport
	for _, item := range r.Items {
		item = AssertRuntimeReportIDAndReset(t, item)
		copyOfReports = append(copyOfReports, item)
	}
	return copyOfReports
}

func CheckFieldsInJSON(t *testing.T, jsonMap map[string]interface{}, mappings map[string]interface{}, excludeFieldList map[string]bool) bool {
	for key, val := range jsonMap {
		if excludeFieldList[key] { // List include id and other object json type
			continue
		}
		switch val.(type) {
		case map[string]interface{}:
			t.Log(key)
			prop := mappings[key].(map[string]interface{})
			if _, ok := prop["properties"]; ok { // this is need to skip map[string][string] where it would be populated as client_labels :{"":""}
				if !CheckFieldsInJSON(t, val.(map[string]interface{}), prop["properties"].(map[string]interface{}), nil) {
					return false
				}
			}
		case []interface{}:
			t.Log(key)
			if !parseArray(t, val.([]interface{}), mappings[key].(map[string]interface{}), excludeFieldList) {
				return false
			}
		default:
			t.Log(key)
			if key == "" || excludeFieldList[key] { // Exclude map values populating key,val as ""
				continue
			}
			if _, ok := mappings[key]; !ok {
				t.Log("Mapping missing the value:", key)
				return false
			}
		}
	}
	return true
}

func IsDynamicMappingDisabled(t *testing.T, mappings map[string]interface{}) {
	require.NotNil(t, mappings["dynamic"])
	require.Equal(t, false, mappings["dynamic"])
}

func parseArray(t *testing.T, anArray []interface{}, mappings map[string]interface{}, excludeFieldList map[string]bool) bool {
	for _, val := range anArray {
		switch val := val.(type) {
		case map[string]interface{}:
			if checkExcludeSliceItem(val, excludeFieldList) {
				continue
			}
			if !CheckFieldsInJSON(t, val, mappings["properties"].(map[string]interface{}), excludeFieldList) {
				return false
			}
		case []interface{}:
			if !parseArray(t, val, mappings, excludeFieldList) {
				return false
			}
		}
	}
	return true
}

func checkExcludeSliceItem(tempMap map[string]interface{}, excludeFieldList map[string]bool) bool {
	for key := range tempMap {
		if excludeFieldList[key] { // if string check for it in excluded list
			return true
		}
	}
	return false
}

func MustUnmarshalStructToMap(t *testing.T, log []byte) map[string]interface{} {
	m := map[string]interface{}{}
	err := json.Unmarshal(log, &m)
	require.NoError(t, err)
	return m
}

func Populate(value reflect.Value) {
	fmt.Println(value.String())
	if value.IsValid() {
		typeOf := value.Type()
		if typeOf.Name() == "Unknown" { // runtime.Unknown is an interface
			return
		}
		if typeOf.Kind() == reflect.Struct {
			for i := 0; i < value.NumField(); i++ {
				f := value.Field(i)
				if f.CanSet() {
					switch f.Kind() {
					case reflect.Interface:
						hack := map[string]interface{}{}
						newMap := reflect.MakeMap(reflect.TypeOf(hack))
						f.Set(newMap)
					case reflect.Map:
						newMap := reflect.MakeMapWithSize(f.Type(), 1)
						key := reflect.Zero(f.Type().Key())
						val := reflect.Zero(f.Type().Elem())
						newMap.SetMapIndex(key, val)
						f.Set(newMap)
					case reflect.Slice:
						newSlice := reflect.MakeSlice(f.Type(), 1, 1)
						f.Set(newSlice)
					case reflect.Struct:
						newStruct := reflect.New(f.Type())
						Populate(newStruct.Elem())
						f.Set(newStruct.Elem())
					case reflect.Ptr:
						newPointer := reflect.New(f.Type().Elem())
						Populate(newPointer.Elem())
						f.Set(newPointer)
					case reflect.String:
						f.SetString("empty")
					case reflect.Bool:
						f.SetBool(true) // when set false omitempty will not populate this field.
					case reflect.Int, reflect.Int64:
						x := int64(7)
						if !f.OverflowInt(x) {
							f.SetInt(x)
						}
					case reflect.Uint64, reflect.Uint8:
						x := uint64(7)
						if !f.OverflowUint(x) {
							f.SetUint(x)
						}
					}
				}
			}
		}
	}
}

func CheckSingleIndexTemplateBootstrapping(t *testing.T, ctx context.Context, client *elastic.Client, idx bapi.Index, i bapi.ClusterInfo, indexPattern, shards, replicas, ILMPolicy string) {
	// Check that the template was created.
	templateExists, err := client.IndexTemplateExists(idx.IndexTemplateName(i)).Do(ctx)
	require.NoError(t, err)
	require.True(t, templateExists)

	// Check that the bootstrap index exists
	indexExists, err := client.IndexExists(idx.BootstrapIndexName(i)).Do(ctx)
	require.NoError(t, err)
	require.True(t, indexExists, "index doesn't exist: %s", idx.BootstrapIndexName(i))

	// Check that write alias exists.
	index := fmt.Sprintf("%s.%s-%s-%s", idx.Name(i), "linseed", time.Now().UTC().Format("20060102"), indexPattern)
	responseAlias, err := client.CatAliases().Do(ctx)
	require.NoError(t, err)
	require.Greater(t, len(responseAlias), 0)
	hasAlias := false
	numWriteIndex := 0
	numNonWriteIndex := 0
	for _, row := range responseAlias {
		if row.Alias == idx.Alias(i) {
			hasAlias = true
			if row.IsWriteIndex == "true" {
				require.Equal(t, index, row.Index)
				numWriteIndex++
			} else {
				require.NotEqual(t, index, row.Index)
				numNonWriteIndex++
			}
		}
	}
	require.True(t, hasAlias)
	require.Equal(t, 1, numWriteIndex)

	responseSettings, err := client.IndexGetSettings(index).Do(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, responseSettings)
	require.Contains(t, responseSettings, index)
	require.NotEmpty(t, responseSettings[index].Settings)
	require.Contains(t, responseSettings[index].Settings, "index")
	settings, _ := responseSettings[index].Settings["index"].(map[string]interface{})
	if idx.HasLifecycleEnabled() {
		// Check lifecycle section
		require.Contains(t, settings, "lifecycle")
		lifecycle, _ := settings["lifecycle"].(map[string]interface{})
		require.Contains(t, lifecycle, "name")
		require.EqualValues(t, lifecycle["name"], ILMPolicy)
		require.EqualValues(t, lifecycle["rollover_alias"], idx.Alias(i))
	}
	// Check shards and replicas
	require.Contains(t, settings, "number_of_replicas")
	require.EqualValues(t, settings["number_of_replicas"], replicas)
	require.Contains(t, settings, "number_of_shards")
	require.EqualValues(t, settings["number_of_shards"], shards)
}
