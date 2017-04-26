package policy

import (
	"context"
	"os"

	"github.com/projectcalico/libcalico-go/lib/client"
	"github.com/prometheus/common/log"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
)

// Storage implements generic apiserver rest.Storage interface
// to help plugin libcalico-go
type Storage struct {
	rest.StandardStorage
	client *client.Client
}

func NewStorage(s rest.StandardStorage) *Storage {
	var err error

	cfg, err := client.LoadClientConfig("")
	if err != nil {
		log.Errorf("Failed to load client config: %q", err)
		os.Exit(1)
	}

	c, err := client.New(*cfg)
	if err != nil {
		log.Errorf("Failed creating client: %q", err)
		os.Exit(1)
	}
	log.Infof("Client: %v", c)

	return &Storage{s, c}
}

func (s *Storage) Create(ctx context.Context, key string, obj, out runtime.Object, ttl uint64) error {

	policy := obj.(*calico.Policy)
	libcalicoPolicy := policy.Spec

	pHandler := s.client.Policies()
	_, err := pHandler.Create(&libcalicoPolicy)
	if err != nil {
		return err
	}

	return nil
}
