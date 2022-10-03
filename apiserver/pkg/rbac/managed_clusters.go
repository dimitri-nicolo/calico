package rbac

import (
	log "github.com/sirupsen/logrus"
	v3 "github.com/tigera/api/pkg/apis/projectcalico/v3"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	rbac_auth "k8s.io/kubernetes/plugin/pkg/auth/authorizer/rbac"
)

// canGetAllManagedClusters determines whether the user is able to get all ManagedClusters.
func (u *userCalculator) canGetAllManagedClusters() bool {
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

	return *u.canGetAllManagedClustersVal
}

// getAllManagedClusters returns the current set of configured ManagedClusters.
func (u *userCalculator) getAllManagedClusters() []string {
	if u.allManagedClusters == nil {
		if managedClusters, err := u.calculator.calicoResourceLister.ListManagedClusters(); err != nil {
			log.WithError(err).Debug("Failed to list ManagedClusters")
			u.errors = append(u.errors, err)
			u.allManagedClusters = make([]string, 0)
		} else {
			for _, managedCluster := range managedClusters {
				u.allManagedClusters = append(u.allManagedClusters, managedCluster.Name)
			}
		}
		log.Debugf("getAllManagedClusters returns %v", u.allManagedClusters)
	}
	return u.allManagedClusters
}

// getGettableManagedClusters determines which ManagedClusters the user is able to get.
func (u *userCalculator) getGettableManagedClusters() []string {
	if u.gettableManagedClusters == nil {
		for _, managedCluster := range u.getAllManagedClusters() {
			if u.canGetAllManagedClusters() || rbac_auth.RulesAllow(authorizer.AttributesRecord{
				Verb:            string(VerbGet),
				APIGroup:        v3.Group,
				Resource:        resourceManagedClusters,
				Name:            managedCluster,
				ResourceRequest: true,
			}, u.getClusterRules()...) {
				u.gettableManagedClusters = append(u.gettableManagedClusters, managedCluster)
			}
		}
		log.Debugf("getGettableManagedClusters returns %v", u.gettableManagedClusters)
	}

	return u.gettableManagedClusters
}
