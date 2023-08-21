// Copyright (c) 2023 Tigera, Inc. All rights reserved.

package waf

import (
	"context"
	"encoding/json"
	"io"
	"os"

	"github.com/olivere/elastic/v7"

	"github.com/projectcalico/calico/linseed/pkg/client"

	v1 "github.com/projectcalico/calico/linseed/pkg/apis/v1"
	"github.com/projectcalico/calico/linseed/pkg/client/rest"
)

var testfiles = []string{
	"test_files/waf_log.json",
}

// WAFLogs implements WAFLogsInterface.
type MockWaf struct {
	restClient rest.RESTClient
	clusterID  string
}

// newWAFLogs returns a new WAFLogsInterface bound to the supplied client.
func newMockWAFLogs(c client.Client, cluster string) client.WAFLogsInterface {
	return &MockWaf{restClient: c.RESTClient(), clusterID: cluster}
}

// List gets the waf for the given input params.
func (f *MockWaf) List(ctx context.Context, params v1.Params) (*v1.List[v1.WAFLog], error) {
	var wafLog v1.WAFLog
	logs := []v1.WAFLog{}
	for _, testfile := range testfiles {
		fileData, err := os.Open(testfile)
		if err != nil {
			panic(err)
		}
		rawLog, err := io.ReadAll(fileData)
		if err != nil {
			panic(err)
		}

		err = json.Unmarshal(rawLog, &wafLog)
		if err != nil {
			panic(err)
		}

		logs = append(logs, wafLog)
	}

	return &v1.List[v1.WAFLog]{Items: logs}, nil
}

// ListInto gets the WAF Logs for the given input params.
func (f *MockWaf) ListInto(ctx context.Context, params v1.Params, l v1.Listable) error {

	return nil
}

// create waf logs
func (f *MockWaf) Create(ctx context.Context, wafl []v1.WAFLog) (*v1.BulkResponse, error) {
	panic("mock Create not implemented")
}

func (f *MockWaf) Aggregations(ctx context.Context, params v1.Params) (elastic.Aggregations, error) {
	return elastic.Aggregations{}, nil
}

// Events implements EventsInterface.
type mockEvents struct {
	restClient rest.RESTClient
	clusterID  string
	events     v1.List[v1.Event]
}

// newEvents returns a new EventsInterface bound to the supplied client.
func newMockEvents(c client.Client, cluster string) client.EventsInterface {
	return &mockEvents{restClient: c.RESTClient(), clusterID: cluster}
}

// List gets the events for the given input params.
func (f *mockEvents) List(ctx context.Context, params v1.Params) (*v1.List[v1.Event], error) {

	return &f.events, nil
}

func (f *mockEvents) Create(ctx context.Context, events []v1.Event) (*v1.BulkResponse, error) {

	return &v1.BulkResponse{}, nil
}

func (f *mockEvents) UpdateDismissFlag(ctx context.Context, events []v1.Event) (*v1.BulkResponse, error) {

	return &v1.BulkResponse{}, nil
}

func (f *mockEvents) Delete(ctx context.Context, events []v1.Event) (*v1.BulkResponse, error) {

	return &v1.BulkResponse{}, nil
}
