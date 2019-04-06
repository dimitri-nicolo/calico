// Copyright 2019 Tigera Inc. All rights reserved.

package calico

import (
	"sync"

	"github.com/tigera/intrusion-detection/controller/pkg/db"

	v3 "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico/v3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type MockGlobalNetworkSetInterface struct {
	GlobalNetworkSet *v3.GlobalNetworkSet
	Error            error
	CreateError      []error
	DeleteError      error
	GetError         error
	UpdateError      error
	WatchError       error
	W                *MockWatch

	m     sync.Mutex
	calls []db.Call
}

func (m *MockGlobalNetworkSetInterface) Create(gns *v3.GlobalNetworkSet) (*v3.GlobalNetworkSet, error) {
	m.m.Lock()
	defer m.m.Unlock()
	m.calls = append(m.calls, db.Call{Method: "Create", GNS: gns.DeepCopy()})
	var err error
	if len(m.CreateError) > 0 {
		err = m.CreateError[0]
		m.CreateError = m.CreateError[1:]
	}
	if err != nil {
		return nil, err
	}
	m.GlobalNetworkSet = gns
	return gns, m.Error
}

func (m *MockGlobalNetworkSetInterface) Update(gns *v3.GlobalNetworkSet) (*v3.GlobalNetworkSet, error) {
	m.m.Lock()
	defer m.m.Unlock()
	m.calls = append(m.calls, db.Call{Method: "Update", GNS: gns.DeepCopy()})
	if m.UpdateError != nil {
		return nil, m.UpdateError
	}
	m.GlobalNetworkSet = gns
	return gns, m.Error
}

func (m *MockGlobalNetworkSetInterface) Delete(name string, options *v1.DeleteOptions) error {
	m.m.Lock()
	defer m.m.Unlock()
	m.calls = append(m.calls, db.Call{Method: "Delete", Name: name})
	return m.DeleteError
}

func (m *MockGlobalNetworkSetInterface) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return m.Error
}

func (m *MockGlobalNetworkSetInterface) Get(name string, options v1.GetOptions) (*v3.GlobalNetworkSet, error) {
	if m.GetError != nil {
		return nil, m.GetError
	}
	return m.GlobalNetworkSet, m.Error
}

func (m *MockGlobalNetworkSetInterface) List(opts v1.ListOptions) (*v3.GlobalNetworkSetList, error) {
	out := &v3.GlobalNetworkSetList{}
	if m.GlobalNetworkSet != nil {
		out.Items = append(out.Items, *m.GlobalNetworkSet)
	}
	return out, m.Error
}

func (m *MockGlobalNetworkSetInterface) Watch(opts v1.ListOptions) (watch.Interface, error) {
	if m.WatchError == nil {
		if m.W == nil {
			m.W = &MockWatch{make(chan watch.Event)}
		}
		return m.W, nil
	} else {
		return nil, m.WatchError
	}
}

func (m *MockGlobalNetworkSetInterface) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v3.GlobalNetworkSet, err error) {
	return nil, m.Error
}

func (m *MockGlobalNetworkSetInterface) Calls() []db.Call {
	var out []db.Call
	m.m.Lock()
	defer m.m.Unlock()
	for _, c := range m.calls {
		out = append(out, c)
	}
	return out
}
