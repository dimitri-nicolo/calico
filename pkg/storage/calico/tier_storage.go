package calico

import (
	"github.com/projectcalico/libcalico-go/lib/client"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
)

type tierStore struct {
	client *client.Client
}

// NewTierStorage creates a new libcalico-based storage.Interface implementation for Tier
func NewTierStorage(opts Options) (storage.Interface, factory.DestroyFunc) {
	return nil, nil
}
