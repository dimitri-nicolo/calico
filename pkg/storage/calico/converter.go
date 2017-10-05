package calico

import (
	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apiv2"
	"github.com/projectcalico/libcalico-go/lib/errors"
	aapi "github.com/tigera/calico-k8sapiserver/pkg/apis/calico"
	"k8s.io/apiserver/pkg/storage"
)

func aapiError(err error, key string) error {
	switch err.(type) {
	case errors.ErrorResourceAlreadyExists:
		return storage.NewKeyExistsError(key, 0)
	case errors.ErrorResourceDoesNotExist:
		return storage.NewKeyNotFoundError(key, 0)
	case errors.ErrorResourceUpdateConflict:
		return storage.NewResourceVersionConflictsError(key, 0)
	default:
		return err
	}
}

func convertToLibcalicoNetworkPolicy(networkPolicy *aapi.NetworkPolicy, libcalicoPolicy *libcalicoapi.NetworkPolicy) {
	libcalicoPolicy.TypeMeta = networkPolicy.TypeMeta
	libcalicoPolicy.ObjectMeta = networkPolicy.ObjectMeta
	libcalicoPolicy.Spec = networkPolicy.Spec
}

func convertToAAPINetworkPolicy(networkPolicy *aapi.NetworkPolicy, libcalicoPolicy *libcalicoapi.NetworkPolicy) {
	networkPolicy.Spec = libcalicoPolicy.Spec
	networkPolicy.TypeMeta = libcalicoPolicy.TypeMeta
	networkPolicy.ObjectMeta = libcalicoPolicy.ObjectMeta
}
