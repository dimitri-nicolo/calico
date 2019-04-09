// Copyright (c) 2019 Tigera, Inc. All rights reserved.
package snapshot

import (
	"time"

	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/tigera/compliance/pkg/list"
	"github.com/tigera/compliance/pkg/resources"
)

var (
	testResourceType = resources.TypeCalicoHostEndpoints
)

type mockSource struct {
	nCalls int
}

func (r *mockSource) RetrieveList(kind metav1.TypeMeta) (*list.TimestampedResourceList, error) {
	if kind != testResourceType {
		panic(fmt.Errorf("unexpected resource type. Got: %s; Expected: %s", kind, testResourceType))
	}

	r.nCalls++
	startSecs := int64(r.nCalls) * 60
	return &list.TimestampedResourceList{
		RequestStartedTimestamp:   metav1.Unix(startSecs, 0),
		RequestCompletedTimestamp: metav1.Unix(startSecs+30, 0),
	}, nil
}

type mockDestination struct {
	nCalls       int
	data         []*list.TimestampedResourceList
	lastListTime time.Time
}

func (r *mockDestination) StoreList(_ metav1.TypeMeta, list *list.TimestampedResourceList) error {
	r.data = append(r.data, list)
	return nil
}

func (r *mockDestination) RetrieveList(tm metav1.TypeMeta, from time.Time) (*list.TimestampedResourceList, error) {
	r.nCalls++
	startSecs := int64(r.nCalls) * 60
	return &list.TimestampedResourceList{
		RequestStartedTimestamp:   metav1.Unix(startSecs, 0),
		RequestCompletedTimestamp: metav1.Unix(startSecs+30, 0),
	}, nil
}

/* TODO(rlb): To add tests.
var _ = Describe("Snapshot", func() {
	It("should call RetrieveList and StoreList a number of times equal to the number of resource types", func() {
		src := &mockSource{}
		dest := &mockDestination{}

		cxt, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(time.Second)
			cancel()
		}()

		Run(cxt, resources.TypeCalicoHostEndpoints, src, dest)

		expectedCalls := len(resources.GetAllResourceHelpers())
		Expect(src.nCalls).To(Equal(expectedCalls))
		Expect(len(dest.data)).To(Equal(expectedCalls))
	})

	It("should call RetrieveList and StoreList a number of times equal to the number of resource types", func() {
		src := &mockSource{}
		dest := &mockDestination{}

		cxt, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(time.Second)
			cancel()
		}()

		Run(cxt, testResourceType, src, dest)

		expectedCalls := len(resources.GetAllResourceHelpers())
		Expect(src.nCalls).To(Equal(expectedCalls))
		Expect(len(dest.data)).To(Equal(expectedCalls))
	})
})
*/
