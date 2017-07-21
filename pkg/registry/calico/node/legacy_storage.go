/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package policy

import (
	"fmt"
	"os"
	"strings"

	"github.com/golang/glog"
	"github.com/projectcalico/libcalico-go/lib/api"
	"github.com/projectcalico/libcalico-go/lib/client"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	"k8s.io/apimachinery/pkg/runtime"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
)

const (
	policyDelim = "."
)

// legacyREST is storage for tiered policies based on libalico-go.
type legacyREST struct {
	store  *genericregistry.Store
	client *client.Client
}

func NewLegacyREST(s *genericregistry.Store) *legacyREST {
	var err error

	cfg, err := client.LoadClientConfig("")
	if err != nil {
		glog.Errorf("Failed to load client config: %q", err)
		os.Exit(1)
	}

	c, err := client.New(*cfg)
	if err != nil {
		glog.Errorf("Failed creating client: %q", err)
		os.Exit(1)
	}
	glog.Infof("Client: %v", c)

	return &legacyREST{s, c}
}

func getTierPolicy(policyName string) (string, string) {
	policySlice := strings.Split(policyName, policyDelim)
	if len(policySlice) < 2 {
		return "default", policySlice[0]
	}
	return policySlice[0], policySlice[1]
}

// Create creates a new version of a resource.
func (l *legacyREST) create(obj runtime.Object) error {
	// Setup the calico Policy for libcalico-go.
	policy := obj.(*calico.Policy)
	libcalicoPolicy := &api.Policy{}
	libcalicoPolicy.Spec = policy.Spec
	libcalicoPolicy.APIVersion = "v1"
	libcalicoPolicy.Kind = "policy"
	tierName, policyName := getTierPolicy(policy.Name)
	libcalicoPolicy.Metadata.Name = policyName
	libcalicoPolicy.Metadata.Tier = tierName

	pHandler := l.client.Policies()
	_, err := pHandler.Create(libcalicoPolicy)
	if err != nil {
		return err
	}

	return nil
}

func (l *legacyREST) get(name string) (*api.Policy, error) {
	tierName, policyName := getTierPolicy(name)
	libcalicoPolicyMD := api.PolicyMetadata{}
	libcalicoPolicyMD.Name = policyName
	libcalicoPolicyMD.Tier = tierName

	pHandler := l.client.Policies()
	policies, err := pHandler.List(libcalicoPolicyMD)
	if err != nil {
		return nil, err
	}
	if len(policies.Items) < 1 {
		return nil, fmt.Errorf("policy %s not found", name)
	}
	return &policies.Items[0], nil
}

func (l *legacyREST) delete(name string) (*api.Policy, error) {
	policy, err := l.get(name)
	if err != nil {
		return nil, err
	}
	tierName, policyName := getTierPolicy(name)
	libcalicoPolicyMD := api.PolicyMetadata{}
	libcalicoPolicyMD.Name = policyName
	libcalicoPolicyMD.Tier = tierName

	pHandler := l.client.Policies()
	err = pHandler.Delete(libcalicoPolicyMD)
	if err != nil {
		return nil, err
	}
	return policy, nil
}

/* TODO
func (l *legacyREST) update(obj runtime.Object) error {

}
*/
