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

package tier

import (
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/projectcalico/libcalico-go/lib/api"
	"github.com/projectcalico/libcalico-go/lib/client"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	"k8s.io/apimachinery/pkg/runtime"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
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

// Create creates a new version of a resource.
func (l *legacyREST) create(obj runtime.Object) error {
	// Setup the calico Policy for libcalico-go.
	tier := obj.(*calico.Tier)
	libcalicoTier := &api.Tier{}
	libcalicoTier.Spec = tier.Spec
	libcalicoTier.APIVersion = "v1"
	libcalicoTier.Kind = "tier"
	libcalicoTier.Metadata.Name = tier.Name

	tHandler := l.client.Tiers()
	_, err := tHandler.Create(libcalicoTier)
	if err != nil {
		return err
	}

	return nil
}

func (l *legacyREST) get(name string) (*api.Tier, error) {
	libcalicoTierMD := api.TierMetadata{}
	libcalicoTierMD.Name = name

	tHandler := l.client.Tiers()
	tiers, err := tHandler.List(libcalicoTierMD)
	if err != nil {
		return nil, err
	}
	if len(tiers.Items) < 1 {
		return nil, fmt.Errorf("tier %s not found", name)
	}
	return &tiers.Items[0], nil
}

func (l *legacyREST) delete(name string) (*api.Tier, error) {
	tier, err := l.get(name)
	if err != nil {
		return nil, err
	}

	libcalicoTierMD := api.TierMetadata{}
	libcalicoTierMD.Name = name

	tHandler := l.client.Tiers()
	err = tHandler.Delete(libcalicoTierMD)
	if err != nil {
		return nil, err
	}
	return tier, nil
}

func (l *legacyREST) getPolicyCount(name string) int {
	libcalicoPolicyMD := api.PolicyMetadata{}
	libcalicoPolicyMD.Name = ""
	libcalicoPolicyMD.Tier = name

	pHandler := l.client.Policies()
	policies, err := pHandler.List(libcalicoPolicyMD)
	if err != nil {
		return 0
	}
	return len(policies.Items)
}

/* TODO
func (l *legacyREST) update(obj runtime.Object) error {

}
*/
