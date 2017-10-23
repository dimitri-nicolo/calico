package calico

import (
	libcalicoapi "github.com/projectcalico/libcalico-go/lib/apis/v2"
	"github.com/projectcalico/libcalico-go/lib/errors"

	aapi "github.com/tigera/calico-k8sapiserver/pkg/apis/projectcalico"
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

func convertToAAPI(libcalicoObject runtime.Object) (res runtime.Object) {
	switch libcalicoObject.(type) {
	case *libcalicoapi.Tier:
		lcgTier := libcalicoObject.(*libcalicoapi.Tier)
		aapiTier := &aapi.Tier{}
		convertToAAPITier(aapiTier, lcgTier)
		return aapiTier
	case *libcalicoapi.NetworkPolicy:
		lcgPolicy := libcalicoObject.(*libcalicoapi.NetworkPolicy)
		aapiPolicy := &aapi.NetworkPolicy{}
		convertToAAPINetworkPolicy(aapiPolicy, lcgPolicy)
		return aapiPolicy
	case *libcalicoapi.GlobalNetworkPolicy:
		lcgPolicy := libcalicoObject.(*libcalicoapi.GlobalNetworkPolicy)
		aapiPolicy := &aapi.GlobalNetworkPolicy{}
		convertToAAPIGlobalNetworkPolicy(aapiPolicy, lcgPolicy)
		return aapiPolicy
	default:
		return nil
	}
}

func convertToLibcalicoNetworkPolicy(networkPolicy *aapi.NetworkPolicy, libcalicoPolicy *libcalicoapi.NetworkPolicy) {
	libcalicoPolicy.TypeMeta = networkPolicy.TypeMeta
	libcalicoPolicy.ObjectMeta = networkPolicy.ObjectMeta
	libcalicoPolicy.Spec = networkPolicy.Spec
}

func convertToAAPINetworkPolicy(networkPolicy *aapi.NetworkPolicy, libcalicoPolicy *libcalicoapi.NetworkPolicy) {
	networkPolicy.Spec = libcalicoPolicy.Spec
	// Tier field maybe left blank when policy created vi OS libcalico.
	// Initialize it to defalt in that case to make work with field selector.
	/*if networkPolicy.Spec.Tier == "" {
		networkPolicy.Spec.Tier = "default"
	}*/
	networkPolicy.TypeMeta = libcalicoPolicy.TypeMeta
	networkPolicy.ObjectMeta = libcalicoPolicy.ObjectMeta
}

func convertToLibcalicoTier(tier *aapi.Tier, libcalicoTier *libcalicoapi.Tier) {
	libcalicoTier.TypeMeta = tier.TypeMeta
	libcalicoTier.ObjectMeta = tier.ObjectMeta
	libcalicoTier.Spec = tier.Spec
}

func convertToAAPITier(tier *aapi.Tier, libcalicoTier *libcalicoapi.Tier) {
	tier.Spec = libcalicoTier.Spec
	tier.TypeMeta = libcalicoTier.TypeMeta
	tier.ObjectMeta = libcalicoTier.ObjectMeta
}

func convertToLibcalicoGlobalNetworkPolicy(globalNetworkPolicy *aapi.GlobalNetworkPolicy, libcalicoPolicy *libcalicoapi.GlobalNetworkPolicy) {
	libcalicoPolicy.TypeMeta = globalNetworkPolicy.TypeMeta
	libcalicoPolicy.ObjectMeta = globalNetworkPolicy.ObjectMeta
	libcalicoPolicy.Spec = globalNetworkPolicy.Spec
}

func convertToAAPIGlobalNetworkPolicy(globalNetworkPolicy *aapi.GlobalNetworkPolicy, libcalicoPolicy *libcalicoapi.GlobalNetworkPolicy) {
	globalNetworkPolicy.Spec = libcalicoPolicy.Spec
	// Tier field maybe left blank when policy created vi OS libcalico.
	// Initialize it to defalt in that case to make work with field selector.
	if globalNetworkPolicy.Spec.Tier == "" {
		globalNetworkPolicy.Spec.Tier = "default"
	}
	globalNetworkPolicy.TypeMeta = libcalicoPolicy.TypeMeta
	globalNetworkPolicy.ObjectMeta = libcalicoPolicy.ObjectMeta
}
