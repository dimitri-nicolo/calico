package rbac

import (
	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	rbac_auth "k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"

	"github.com/projectcalico/calico/apiserver/pkg/storage/calico"
)

// canGetAllManagedClusters determines whether the user is able to get all ManagedClusters.
func (u *userCalculator) canGetAllManagedClusters() bool {
	if calico.MultiTenantEnabled {
		// In multi-tenant management clusters, nobody should have access to all ManagedClusters - we can
		// short-circuit the calculation and return false.
		return false
	}

	if u.canGetAllManagedClustersVal == nil {
		allManagedClusters := rbac_auth.RulesAllow(authorizer.AttributesRecord{
			Verb:            string(VerbGet),
			APIGroup:        v3.Group,
			Resource:        resourceManagedClusters,
			Name:            "",
			ResourceRequest: true,
		}, u.getClusterRules()...)
		u.canGetAllManagedClustersVal = &allManagedClusters
	}

	log.Debugf("Can get all managed clusters? %t", *u.canGetAllManagedClustersVal)
	return *u.canGetAllManagedClustersVal
}

// getAllManagedClusters returns the current set of configured ManagedClusters.
func (u *userCalculator) getAllManagedClusters() []types.NamespacedName {
	if u.allManagedClusters == nil {
		if managedClusters, err := u.calculator.calicoResourceLister.ListManagedClusters(); err != nil {
			log.WithError(err).Debug("Failed to list ManagedClusters")
			u.errors = append(u.errors, err)
			u.allManagedClusters = make([]types.NamespacedName, 0)
		} else {
			for _, managedCluster := range managedClusters {
				u.allManagedClusters = append(u.allManagedClusters, types.NamespacedName{Name: managedCluster.Name, Namespace: managedCluster.Namespace})
			}
		}
		log.Debugf("getAllManagedClusters returns %v", u.allManagedClusters)
	}
	return u.allManagedClusters
}

// getGettableManagedClusters determines which ManagedClusters the user is able to get.
func (u *userCalculator) getGettableManagedClusters() []types.NamespacedName {
	if u.gettableManagedClusters == nil {
		for _, managedCluster := range u.getAllManagedClusters() {
			// Get all cluster-scoped rules.
			rules := u.getClusterRules()
			if managedCluster.Namespace != "" {
				// Include namespace-scoped rules if the ManagedCluster is namespaced.
				rules = append(rules, u.getNamespacedRules()[managedCluster.Namespace]...)
			}
			if u.canGetAllManagedClusters() || rbac_auth.RulesAllow(authorizer.AttributesRecord{
				Verb:            string(VerbGet),
				APIGroup:        v3.Group,
				Resource:        resourceManagedClusters,
				Name:            managedCluster.Name,
				Namespace:       managedCluster.Namespace,
				ResourceRequest: true,
			}, rules...) {
				u.gettableManagedClusters = append(u.gettableManagedClusters, managedCluster)
			}
		}
		log.Debugf("getGettableManagedClusters returns %v", u.gettableManagedClusters)
	}

	return u.gettableManagedClusters
}
