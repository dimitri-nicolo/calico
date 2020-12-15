// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package authorization

import (
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/mock"

	"github.com/projectcalico/kube-controllers/pkg/elasticsearch"
	"github.com/projectcalico/kube-controllers/pkg/rbaccache"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("synchronizeRoleMappings", func() {
	Context("Update ClusterRole", func() {
		DescribeTable(
			"ClusterRole rule conversion to elasticsearch role mapping",
			func(rules []rbacv1.PolicyRule, expectedRoleMapping elasticsearch.RoleMapping) {
				clusterRole := &rbacv1.ClusterRole{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-resource",
					},
				}

				mockESCLI := elasticsearch.NewMockClient()
				// Verify that the correct role mapping is created given whats returned from the cache
				mockESCLI.On("CreateRoleMapping", expectedRoleMapping).Return(nil)

				mockClusterRoleCache := rbaccache.NewMockClusterRoleCache()
				mockClusterRoleCache.On("AddClusterRole", clusterRole).Return(true)
				mockClusterRoleCache.On("ClusterRoleSubjects", clusterRole.Name, rbacv1.UserKind).Return([]rbacv1.Subject{{
					Kind: rbacv1.UserKind,
					Name: "user@test.com",
				}})
				mockClusterRoleCache.On("ClusterRoleSubjects", clusterRole.Name, rbacv1.GroupKind).Return([]rbacv1.Subject{{
					Kind: rbacv1.GroupKind,
					Name: "testgroup",
				}})
				// Return all the valid resource names so we can test the conversion or resource names to elasticsearch role names
				mockClusterRoleCache.On("ClusterRoleRules", mock.Anything).Return(rules)

				resourceUpdates := make(chan resourceUpdate)

				synchronizer := esRoleMappingSynchronizer{
					esCLI:           mockESCLI,
					roleCache:       mockClusterRoleCache,
					resourceUpdates: resourceUpdates,
				}

				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()
					synchronizer.synchronizeRoleMappings()
				}()

				resourceUpdates <- resourceUpdate{
					typ:      resourceUpdated,
					name:     clusterRole.Name,
					resource: clusterRole,
				}

				close(resourceUpdates)

				wg.Wait()

				mockESCLI.AssertExpectations(GinkgoT())
			},
			TableEntry{
				Description: "The ClusterRole resource is *",
				Parameters: []interface{}{
					[]rbacv1.PolicyRule{{
						APIGroups:     []string{"lma.tigera.io"},
						ResourceNames: []string{"flows", "audit*", "audit_ee", "audit_kube", "events", "dns", "l7", "kibana_login", "kibana_admin", "elasticsearch_superuser"},
						Resources:     []string{"*"},
					}},
					elasticsearch.RoleMapping{
						Name: "tigera-k8s-test-resource",
						Roles: []string{"flows_viewer", "audit_viewer", "audit_ee_viewer",
							"audit_kube_viewer", "events_viewer", "dns_viewer", "l7_viewer", "kibana_viewer", "kibana_admin", "superuser",
						},
						Rules: map[string][]elasticsearch.Rule{
							"any": {
								{
									Field: map[string]string{
										"username": "user@test.com",
									},
								},
								{
									Field: map[string]string{
										"groups": "testgroup",
									},
								},
							},
						},
						Enabled: true,
					},
				},
			},
			TableEntry{
				Description: "The ClusterRole resource is a specific list of clusters",
				Parameters: []interface{}{
					[]rbacv1.PolicyRule{{
						APIGroups:     []string{"lma.tigera.io"},
						ResourceNames: []string{"flows", "audit*", "audit_ee", "audit_kube", "events", "dns", "l7", "kibana_login", "kibana_admin", "elasticsearch_superuser"},
						Resources:     []string{"cluster_1", "cluster_2"},
					}},
					elasticsearch.RoleMapping{
						Name: "tigera-k8s-test-resource",
						Roles: []string{
							"flows_viewer_cluster_1", "audit_viewer_cluster_1", "audit_ee_viewer_cluster_1", "audit_kube_viewer_cluster_1",
							"events_viewer_cluster_1", "dns_viewer_cluster_1", "l7_viewer_cluster_1", "flows_viewer_cluster_2", "audit_viewer_cluster_2",
							"audit_ee_viewer_cluster_2", "audit_kube_viewer_cluster_2", "events_viewer_cluster_2", "dns_viewer_cluster_2",
							"l7_viewer_cluster_2", "kibana_viewer", "kibana_admin", "superuser",
						},
						Rules: map[string][]elasticsearch.Rule{
							"any": {
								{
									Field: map[string]string{
										"username": "user@test.com",
									},
								},
								{
									Field: map[string]string{
										"groups": "testgroup",
									},
								},
							},
						},
						Enabled: true,
					},
				},
			},
			TableEntry{
				Description: "The ClusterRole has no resource names",
				Parameters: []interface{}{
					[]rbacv1.PolicyRule{{
						APIGroups: []string{"lma.tigera.io"},
						Resources: []string{"*"},
					}},
					elasticsearch.RoleMapping{
						Name:  "tigera-k8s-test-resource",
						Roles: []string{"flows_viewer", "audit_viewer", "events_viewer", "dns_viewer", "l7_viewer"},
						Rules: map[string][]elasticsearch.Rule{
							"any": {
								{
									Field: map[string]string{
										"username": "user@test.com",
									},
								},
								{
									Field: map[string]string{
										"groups": "testgroup",
									},
								},
							},
						},
						Enabled: true,
					},
				},
			},
		)
	})

	Context("Update ClusterRoleBinding", func() {
		It("Triggers a role synchronization when the ClusterRoleBinding is added", func() {
			clusterRoleBinding := &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-binding",
				},
				RoleRef: rbacv1.RoleRef{
					Name: "test-role",
				},
			}

			mockESCLI := elasticsearch.NewMockClient()
			mockESCLI.On("CreateRoleMapping", mock.Anything).Return(nil)

			mockClusterRoleCache := rbaccache.NewMockClusterRoleCache()
			mockClusterRoleCache.On("AddClusterRoleBinding", clusterRoleBinding).Return(true)
			mockClusterRoleCache.On("ClusterRoleSubjects", clusterRoleBinding.RoleRef.Name, rbacv1.UserKind).Return([]rbacv1.Subject{{
				Kind: rbacv1.UserKind,
				Name: "user@test.com",
			}})
			mockClusterRoleCache.On("ClusterRoleSubjects", clusterRoleBinding.RoleRef.Name, rbacv1.GroupKind).Return([]rbacv1.Subject{{
				Kind: rbacv1.GroupKind,
				Name: "testgroup",
			}})
			// Return all the valid resource names so we can test the conversion or resource names to elasticsearch role names
			mockClusterRoleCache.On("ClusterRoleRules", mock.Anything).Return([]rbacv1.PolicyRule{{
				APIGroups:     []string{"lma.tigera.io"},
				ResourceNames: []string{"flows", "audit*", "audit_ee", "audit_kube", "events", "dns", "kibana_login", "elasticsearch_superuser"},
				Resources:     []string{"*"},
			}})

			resourceUpdates := make(chan resourceUpdate)

			synchronizer := esRoleMappingSynchronizer{
				esCLI:           mockESCLI,
				roleCache:       mockClusterRoleCache,
				resourceUpdates: resourceUpdates,
			}

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				synchronizer.synchronizeRoleMappings()
			}()

			resourceUpdates <- resourceUpdate{
				typ:      resourceUpdated,
				name:     clusterRoleBinding.Name,
				resource: clusterRoleBinding,
			}

			close(resourceUpdates)

			wg.Wait()

			mockESCLI.AssertExpectations(GinkgoT())
		})
	})

	Context("Delete ClusterRole", func() {
		clusterRoleName := "test-cluster-role"
		mockESCLI := elasticsearch.NewMockClient()
		// Verify that the correct role mapping is created given whats returned from the cache
		mockESCLI.On("DeleteRoleMapping", "tigera-k8s-test-cluster-role").Return(true, nil)

		mockClusterRoleCache := rbaccache.NewMockClusterRoleCache()
		mockClusterRoleCache.On("RemoveClusterRole", clusterRoleName).Return(true)

		mockClusterRoleCache.On("ClusterRoleSubjects", clusterRoleName, rbacv1.UserKind).Return([]rbacv1.Subject{})
		mockClusterRoleCache.On("ClusterRoleSubjects", clusterRoleName, rbacv1.GroupKind).Return([]rbacv1.Subject{})
		// Return all the valid resource names so we can test the conversion or resource names to elasticsearch role names
		mockClusterRoleCache.On("ClusterRoleRules", mock.Anything).Return([]rbacv1.PolicyRule{})

		resourceUpdates := make(chan resourceUpdate)

		synchronizer := esRoleMappingSynchronizer{
			esCLI:           mockESCLI,
			roleCache:       mockClusterRoleCache,
			resourceUpdates: resourceUpdates,
		}

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			synchronizer.synchronizeRoleMappings()
		}()

		var clusterRole *rbacv1.ClusterRole
		resourceUpdates <- resourceUpdate{
			typ:      resourceDeleted,
			name:     clusterRoleName,
			resource: clusterRole,
		}

		close(resourceUpdates)

		wg.Wait()

		mockESCLI.AssertExpectations(GinkgoT())
	})

	Context("Delete ClusterRoleBinding", func() {
		clusterRoleBindingName := "test-cluster-role-binding"
		clusterRoleName := "test-cluster-role"

		mockESCLI := elasticsearch.NewMockClient()
		// Verify that the correct role mapping is created given whats returned from the cache
		mockESCLI.On("DeleteRoleMapping", "tigera-k8s-test-cluster-role").Return(true, nil)

		mockClusterRoleCache := rbaccache.NewMockClusterRoleCache()
		mockClusterRoleCache.On("RemoveClusterRoleBinding", clusterRoleBindingName).Return(true)
		mockClusterRoleCache.On("ClusterRoleNameForBinding", clusterRoleBindingName).Return(clusterRoleName)

		mockClusterRoleCache.On("ClusterRoleSubjects", clusterRoleName, rbacv1.UserKind).Return([]rbacv1.Subject{})
		mockClusterRoleCache.On("ClusterRoleSubjects", clusterRoleName, rbacv1.GroupKind).Return([]rbacv1.Subject{})
		// Return all the valid resource names so we can test the conversion or resource names to elasticsearch role names
		mockClusterRoleCache.On("ClusterRoleRules", mock.Anything).Return([]rbacv1.PolicyRule{})

		resourceUpdates := make(chan resourceUpdate)

		synchronizer := esRoleMappingSynchronizer{
			esCLI:           mockESCLI,
			roleCache:       mockClusterRoleCache,
			resourceUpdates: resourceUpdates,
		}

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			synchronizer.synchronizeRoleMappings()
		}()

		var clusterRoleBinding *rbacv1.ClusterRoleBinding
		resourceUpdates <- resourceUpdate{
			typ:      resourceDeleted,
			name:     clusterRoleBindingName,
			resource: clusterRoleBinding,
		}

		close(resourceUpdates)

		wg.Wait()

		mockESCLI.AssertExpectations(GinkgoT())
	})

	Context("claim prefixes", func() {
		It("usernamePrefix and groupPrefix are stripped from user and group names before mappings are created for them", func() {
			clusterRole := &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-resource",
				},
			}

			mockESCLI := elasticsearch.NewMockClient()
			// Verify that the correct role mapping is created given whats returned from the cache
			mockESCLI.On("CreateRoleMapping", elasticsearch.RoleMapping{
				Name:  "tigera-k8s-test-resource",
				Roles: []string{"flows_viewer", "audit_viewer", "events_viewer", "dns_viewer", "l7_viewer"},
				Rules: map[string][]elasticsearch.Rule{
					"any": {
						{
							Field: map[string]string{
								"username": "user@test.com",
							},
						},
						{
							Field: map[string]string{
								"groups": "testgroup",
							},
						},
					},
				},
				Enabled: true,
			}).Return(nil)

			mockClusterRoleCache := rbaccache.NewMockClusterRoleCache()
			mockClusterRoleCache.On("AddClusterRole", clusterRole).Return(true)
			mockClusterRoleCache.On("ClusterRoleSubjects", clusterRole.Name, rbacv1.UserKind).Return([]rbacv1.Subject{{
				Kind: rbacv1.UserKind,
				Name: "oidc:user@test.com",
			}})
			mockClusterRoleCache.On("ClusterRoleSubjects", clusterRole.Name, rbacv1.GroupKind).Return([]rbacv1.Subject{{
				Kind: rbacv1.GroupKind,
				Name: "oidc:testgroup",
			}})
			// Return all the valid resource names so we can test the conversion or resource names to elasticsearch role names
			mockClusterRoleCache.On("ClusterRoleRules", mock.Anything).Return([]rbacv1.PolicyRule{{
				APIGroups: []string{"lma.tigera.io"},
				Resources: []string{"*"},
			}})

			resourceUpdates := make(chan resourceUpdate)

			synchronizer := esRoleMappingSynchronizer{
				esCLI:           mockESCLI,
				roleCache:       mockClusterRoleCache,
				resourceUpdates: resourceUpdates,
				usernamePrefix:  "oidc:",
				groupPrefix:     "oidc:",
			}

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				synchronizer.synchronizeRoleMappings()
			}()

			resourceUpdates <- resourceUpdate{
				typ:      resourceUpdated,
				name:     clusterRole.Name,
				resource: clusterRole,
			}

			close(resourceUpdates)

			wg.Wait()

			mockESCLI.AssertExpectations(GinkgoT())
		})
	})

	Context("removeStaleMappings", func() {
		It("Deletes an Elasticsearch role mapping with a proper associated ClusterRole", func() {
			mockESCLI := elasticsearch.NewMockClient()
			// Verify that the correct role mapping is created given whats returned from the cache
			mockESCLI.On("GetRoleMappings").Return([]elasticsearch.RoleMapping{
				{Name: "tigera-k8s-test-1-cluster-role"},
				{Name: "tigera-k8s-test-2-cluster-role"},
			}, nil)
			mockESCLI.On("DeleteRoleMapping", "tigera-k8s-test-1-cluster-role").Return(true, nil)

			mockClusterRoleCache := rbaccache.NewMockClusterRoleCache()
			mockClusterRoleCache.On("ClusterRoleNamesWithBindings").Return([]string{"test-2-cluster-role"})

			resourceUpdates := make(chan resourceUpdate)

			synchronizer := esRoleMappingSynchronizer{
				esCLI:           mockESCLI,
				roleCache:       mockClusterRoleCache,
				resourceUpdates: resourceUpdates,
			}

			Expect(synchronizer.removeStaleMappings()).ShouldNot(HaveOccurred())

			mockESCLI.AssertExpectations(GinkgoT())
		})
	})
})
