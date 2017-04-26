package policy

import (
	"os"

	"github.com/projectcalico/libcalico-go/lib/api"
	"github.com/projectcalico/libcalico-go/lib/client"
	"github.com/prometheus/common/log"
	"github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
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

// Create inserts a new item according to the unique key from the object.
func (s *Storage) Create(ctx genericapirequest.Context, obj runtime.Object) (runtime.Object, error) {

	policy := obj.(*calico.Policy)
	libcalicoPolicy := policy.Spec

	pHandler := s.client.Policies()
	p, err := pHandler.Create(&libcalicoPolicy)
	if err != nil {
		return nil, err
	}

	pOut := calico.Policy{Spec: *p}
	return &pOut, nil
}

// Get retrieves the item from storage.
func (s *Storage) Get(ctx genericapirequest.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {

	policyMetadata := api.PolicyMetadata{Name: name}
	pHandler := s.client.Policies()
	policyList, err := pHandler.List(policyMetadata)
	if err != nil {
		return nil, err
	}

	pListOut := calico.PolicyList{Items: []calico.Policy{}}
	for _, item := range policyList.Items {
		pOut := calico.Policy{Spec: item}
		pListOut.Items = append(pListOut.Items, pOut)
	}

	return &pListOut, nil
}
