// Copyright (c) 2018-2019 Tigera, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package infrastructure

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	log "github.com/sirupsen/logrus"

	"github.com/projectcalico/calico/libcalico-go/lib/apiconfig"
)

type InfraFactory func() DatastoreInfra

// DatastoreDescribe is a replacement for ginkgo.Describe which invokes Describe
// multiple times for one or more different datastore drivers - passing in the
// function to retrieve the appropriate datastore infrastructure.  This allows
// easy construction of end-to-end tests covering multiple different datastore
// drivers.
//
// The *datastores* parameter is a slice of the DatastoreTypes to test.
func DatastoreDescribe(description string, datastores []apiconfig.DatastoreType, body func(InfraFactory)) bool {
	for _, ds := range datastores {
		switch ds {
		case apiconfig.EtcdV3:
			if len(datastores) > 1 {
				// Enterprise only supports KDD so skip running etcd tests if this test also runs
				// on KDD.
				log.Infof("Skipping etcd mode tests for %q", description)
				continue
			}
			Describe(fmt.Sprintf("%s (etcdv3 backend)", description),
				func() {
					body(createEtcdDatastoreInfra)
				})
		case apiconfig.Kubernetes:
			Describe(fmt.Sprintf("%s (kubernetes backend)", description),
				func() {
					body(createLocalK8sDatastoreInfra)
				})
		default:
			panic(fmt.Errorf("Unknown DatastoreType, %s", ds))
		}
	}

	return true
}

type LocalRemoteInfraFactories struct {
	Local  InfraFactory
	Remote InfraFactory
}

func (r *LocalRemoteInfraFactories) IsRemoteSetup() bool {
	return r.Remote != nil
}
func (r *LocalRemoteInfraFactories) AllFactories() []InfraFactory {
	factories := []InfraFactory{r.Local}
	if r.IsRemoteSetup() {
		factories = append(factories, r.Remote)
	}
	return factories
}

// DatastoreDescribeWithRemote is similar to DatastoreDescribe. It invokes Describe for the provided datastores, providing
// just a local datastore driver. However, it also invokes Describe for supported remote scenarios, providing both a local
// and remote datastore drivers. Currently, the only remote scenario is local kubernetes and remote kubernetes.
func DatastoreDescribeWithRemote(description string, localDatastores []apiconfig.DatastoreType, body func(factories LocalRemoteInfraFactories)) bool {
	for _, ds := range localDatastores {
		switch ds {
		case apiconfig.EtcdV3:
			if len(localDatastores) > 1 {
				// Enterprise only supports KDD so skip running etcd tests if this test also runs
				// on KDD.
				log.Infof("Skipping etcd mode tests for %q", description)
				continue
			}
			Describe(fmt.Sprintf("%s (etcdv3 backend)", description),
				func() {
					body(LocalRemoteInfraFactories{Local: createEtcdDatastoreInfra})
				})
		case apiconfig.Kubernetes:
			Describe(fmt.Sprintf("%s (kubernetes backend)", description),
				func() {
					body(LocalRemoteInfraFactories{Local: createLocalK8sDatastoreInfra})
				})
		default:
			panic(fmt.Errorf("Unknown DatastoreType, %s", ds))
		}
	}

	Describe(fmt.Sprintf("%s (local kubernetes, remote kubernetes)", description),
		func() {
			body(LocalRemoteInfraFactories{Local: createLocalK8sDatastoreInfra, Remote: createRemoteK8sDatastoreInfra})
		})

	return true
}
