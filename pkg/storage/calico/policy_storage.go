package calico

import (
	"github.com/projectcalico/libcalico-go/lib/client"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
)

type policyStore struct {
	client *client.Client
}

// NewPolicyStorage creates a new libcalico-based storage.Interface implementation for Policy
func NewPolicyStorage(opts Options) (storage.Interface, factory.DestroyFunc) {

}
