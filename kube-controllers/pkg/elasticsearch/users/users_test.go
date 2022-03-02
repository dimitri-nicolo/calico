// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

package users_test

import (
	"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch"
	"github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch/users"
)

var _ = Describe("ElasticseachUsers", func() {
	Context("management flag set to false", func() {
		It("the expected users and roles are available", func() {
			privateUsers, publicUsers := users.ElasticsearchUsers("managed-cluster", false)
			testElasticsearchUsers(privateUsers, publicUsers,
				map[users.ElasticsearchUserName]elasticsearch.User{
					"tigera-fluentd": {
						Username: "tigera-fluentd-managed-cluster-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-fluentd-managed-cluster-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates", "manage_ilm"},
								Indices: []elasticsearch.RoleIndex{{
									Names:      []string{"tigera_secure_ee_*.managed-cluster.*"},
									Privileges: []string{"create_index", "write", "manage"},
								}},
							},
						}},
					},
					"tigera-eks-log-forwarder": {
						Username: "tigera-eks-log-forwarder-managed-cluster-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-eks-log-forwarder-managed-cluster-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates", "manage_ilm"},
								Indices: []elasticsearch.RoleIndex{{
									Names:      []string{"tigera_secure_ee_audit_kube.managed-cluster.*"},
									Privileges: []string{"create_index", "read", "write", "manage"},
								}},
							},
						}},
					},
					"tigera-ee-compliance-benchmarker": {
						Username: "tigera-ee-compliance-benchmarker-managed-cluster-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-compliance-benchmarker-managed-cluster-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates"},
								Indices: []elasticsearch.RoleIndex{{
									Names:      []string{"tigera_secure_ee_benchmark_results.managed-cluster.*"},
									Privileges: []string{"create_index", "write", "view_index_metadata", "read", "manage"},
								}},
							},
						}},
					},
					"tigera-ee-compliance-controller": {
						Username: "tigera-ee-compliance-controller-managed-cluster-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-compliance-controller-managed-cluster-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates"},
								Indices: []elasticsearch.RoleIndex{{
									Names:      []string{"tigera_secure_ee_compliance_reports.managed-cluster.*"},
									Privileges: []string{"read"},
								}},
							},
						}},
					},
					"tigera-ee-compliance-reporter": {
						Username: "tigera-ee-compliance-reporter-managed-cluster-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-compliance-reporter-managed-cluster-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates"},
								Indices: []elasticsearch.RoleIndex{
									{
										Names:      []string{"tigera_secure_ee_audit_*.managed-cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_snapshots.managed-cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_benchmark_results.managed-cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_flows.managed-cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_compliance_reports.managed-cluster.*"},
										Privileges: []string{"create_index", "write", "view_index_metadata", "read", "manage"},
									},
								},
							},
						}},
					},
					"tigera-ee-compliance-snapshotter": {
						Username: "tigera-ee-compliance-snapshotter-managed-cluster-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-compliance-snapshotter-managed-cluster-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates"},
								Indices: []elasticsearch.RoleIndex{{
									Names:      []string{"tigera_secure_ee_snapshots.managed-cluster.*"},
									Privileges: []string{"create_index", "write", "view_index_metadata", "read", "manage"},
								}},
							},
						}},
					},
					"tigera-ee-intrusion-detection": {
						Username: "tigera-ee-intrusion-detection-managed-cluster-secure",
						Roles: []elasticsearch.Role{
							{
								Name: "tigera-ee-intrusion-detection-managed-cluster-secure",
								Definition: &elasticsearch.RoleDefinition{
									Cluster: []string{"monitor", "manage_index_templates"},
									Indices: []elasticsearch.RoleIndex{
										{
											Names:      []string{"tigera_secure_ee_*.managed-cluster.*"},
											Privileges: []string{"read"},
										},
										{
											Names: []string{
												".tigera.ipset.managed-cluster",
												".tigera.domainnameset.managed-cluster",
												".tigera.forwarderconfig.managed-cluster",
												"tigera_secure_ee_events.managed-cluster*",
											},
											Privileges: []string{"all"},
										},
									},
								},
							},
							{
								Name: "watcher_admin",
							},
						},
					},
					"tigera-ee-ad-job": {
						Username: "tigera-ee-ad-job-managed-cluster-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-ad-job-managed-cluster-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates"},
								Indices: []elasticsearch.RoleIndex{
									{
										Names:      []string{"tigera_secure_ee_flows.managed-cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_dns.managed-cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_l7.managed-cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_runtime.managed-cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_events.managed-cluster.*"},
										Privileges: []string{"read", "write"},
									},
								},
							},
						}},
					},
					"tigera-ee-performance-hotspots": {
						Username: "tigera-ee-performance-hotspots-managed-cluster-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-performance-hotspots-managed-cluster-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates"},
								Indices: []elasticsearch.RoleIndex{
									{
										Names:      []string{"tigera_secure_ee_flows.managed-cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_dns.managed-cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_l7.managed-cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_events.managed-cluster.*"},
										Privileges: []string{"read", "write"},
									},
								},
							},
						}},
					},
				},
				map[users.ElasticsearchUserName]elasticsearch.User{
					"tigera-fluentd": {
						Username: "tigera-fluentd-managed-cluster",
					},
					"tigera-eks-log-forwarder": {
						Username: "tigera-eks-log-forwarder-managed-cluster",
					},
					"tigera-ee-compliance-benchmarker": {
						Username: "tigera-ee-compliance-benchmarker-managed-cluster",
					},
					"tigera-ee-compliance-controller": {
						Username: "tigera-ee-compliance-controller-managed-cluster",
					},
					"tigera-ee-compliance-reporter": {
						Username: "tigera-ee-compliance-reporter-managed-cluster",
					},
					"tigera-ee-compliance-snapshotter": {
						Username: "tigera-ee-compliance-snapshotter-managed-cluster",
					},
					"tigera-ee-intrusion-detection": {
						Username: "tigera-ee-intrusion-detection-managed-cluster",
					},
					"tigera-ee-ad-job": {
						Username: "tigera-ee-ad-job-managed-cluster",
					},
					"tigera-ee-performance-hotspots": {
						Username: "tigera-ee-performance-hotspots-managed-cluster",
					},
				},
			)
		})
	})
	Context("management flag set to true", func() {
		It("the expected users and roles are available", func() {
			privateUsers, publicUsers := users.ElasticsearchUsers("cluster", true)
			testElasticsearchUsers(privateUsers, publicUsers,
				map[users.ElasticsearchUserName]elasticsearch.User{
					"tigera-fluentd": {
						Username: "tigera-fluentd-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-fluentd-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates", "manage_ilm"},
								Indices: []elasticsearch.RoleIndex{{
									Names:      []string{"tigera_secure_ee_*.cluster.*"},
									Privileges: []string{"create_index", "write", "manage"},
								}},
							},
						}},
					},
					"tigera-eks-log-forwarder": {
						Username: "tigera-eks-log-forwarder-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-eks-log-forwarder-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates", "manage_ilm"},
								Indices: []elasticsearch.RoleIndex{{
									Names:      []string{"tigera_secure_ee_audit_kube.cluster.*"},
									Privileges: []string{"create_index", "read", "write", "manage"},
								}},
							},
						}},
					},
					"tigera-ee-compliance-benchmarker": {
						Username: "tigera-ee-compliance-benchmarker-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-compliance-benchmarker-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates"},
								Indices: []elasticsearch.RoleIndex{{
									Names:      []string{"tigera_secure_ee_benchmark_results.cluster.*"},
									Privileges: []string{"create_index", "write", "view_index_metadata", "read", "manage"},
								}},
							},
						}},
					},
					"tigera-ee-compliance-controller": {
						Username: "tigera-ee-compliance-controller-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-compliance-controller-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates"},
								Indices: []elasticsearch.RoleIndex{{
									Names:      []string{"tigera_secure_ee_compliance_reports.cluster.*"},
									Privileges: []string{"read"},
								}},
							},
						}},
					},
					"tigera-ee-compliance-reporter": {
						Username: "tigera-ee-compliance-reporter-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-compliance-reporter-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates"},
								Indices: []elasticsearch.RoleIndex{
									{
										Names:      []string{"tigera_secure_ee_audit_*.cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_snapshots.cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_benchmark_results.cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_flows.cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_compliance_reports.cluster.*"},
										Privileges: []string{"create_index", "write", "view_index_metadata", "read", "manage"},
									},
								},
							},
						}},
					},
					"tigera-ee-compliance-snapshotter": {
						Username: "tigera-ee-compliance-snapshotter-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-compliance-snapshotter-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates"},
								Indices: []elasticsearch.RoleIndex{{
									Names:      []string{"tigera_secure_ee_snapshots.cluster.*"},
									Privileges: []string{"create_index", "write", "view_index_metadata", "read", "manage"},
								}},
							},
						}},
					},
					"tigera-ee-intrusion-detection": {
						Username: "tigera-ee-intrusion-detection-secure",
						Roles: []elasticsearch.Role{
							{
								Name: "tigera-ee-intrusion-detection-secure",
								Definition: &elasticsearch.RoleDefinition{
									Cluster: []string{"monitor", "manage_index_templates"},
									Indices: []elasticsearch.RoleIndex{
										{
											Names:      []string{"tigera_secure_ee_*.cluster.*"},
											Privileges: []string{"read"},
										},
										{
											Names:      []string{"tigera_secure_ee_flows.*.*"},
											Privileges: []string{"read"},
										},
										{
											Names:      []string{"tigera_secure_ee_audit_*.*.*"},
											Privileges: []string{"read"},
										},
										{
											Names:      []string{"tigera_secure_ee_dns.*.*"},
											Privileges: []string{"read"},
										},
										{
											Names: []string{
												".tigera.ipset.cluster",
												".tigera.domainnameset.cluster",
												".tigera.forwarderconfig.cluster",
												"tigera_secure_ee_events.*",
											},
											Privileges: []string{"all"},
										},
									},
								},
							},
							{
								Name: "watcher_admin",
							},
						},
					},
					"tigera-ee-installer": {
						Username: "tigera-ee-installer-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-installer-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"manage_watcher", "manage"},
								Indices: []elasticsearch.RoleIndex{
									{
										Names:      []string{"tigera_secure_ee_*.cluster.*", "tigera_secure_ee_events.cluster.*"},
										Privileges: []string{"read", "write"},
									},
								},
								Applications: []elasticsearch.Application{{
									Application: "kibana-.kibana",
									Privileges:  []string{"all"},
									Resources:   []string{"*"},
								}},
							},
						}},
					},
					"tigera-ee-ad-job": {
						Username: "tigera-ee-ad-job-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-ad-job-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates"},
								Indices: []elasticsearch.RoleIndex{
									{
										Names:      []string{"tigera_secure_ee_flows.*.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_dns.*.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_l7.*.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_runtime.*.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_events.*.*"},
										Privileges: []string{"read", "write"},
									},
								},
							},
						}},
					},
					"tigera-ee-sasha": {
						Username: "tigera-ee-sasha-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-sasha-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates"},
								Indices: []elasticsearch.RoleIndex{
									{
										Names:      []string{"tigera_secure_ee_runtime.*.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_events.*.*"},
										Privileges: []string{"read", "write"},
									},
								},
							},
						}},
					},
					"tigera-ee-performance-hotspots": {
						Username: "tigera-ee-performance-hotspots-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-performance-hotspots-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates"},
								Indices: []elasticsearch.RoleIndex{
									{
										Names:      []string{"tigera_secure_ee_flows.cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_dns.cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_l7.cluster.*"},
										Privileges: []string{"read"},
									},
									{
										Names:      []string{"tigera_secure_ee_events.cluster.*"},
										Privileges: []string{"read", "write"},
									},
								},
							},
						}},
					},
					"tigera-ee-compliance-server": {
						Username: "tigera-ee-compliance-server-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-compliance-server-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates"},
								Indices: []elasticsearch.RoleIndex{{
									Names:      []string{"tigera_secure_ee_compliance_reports.*.*"},
									Privileges: []string{"read"},
								}},
							},
						}},
					},
					"tigera-ee-manager": {
						Username: "tigera-ee-manager-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-manager-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor"},
								Indices: []elasticsearch.RoleIndex{{
									Names:      []string{"tigera_secure_ee_*.*.*", "tigera_secure_ee_events.*", ".kibana"},
									Privileges: []string{"read"},
								}},
							},
						}},
					},
					"tigera-ee-curator": {
						Username: "tigera-ee-curator-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-curator-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates"},
								Indices: []elasticsearch.RoleIndex{{
									// Curator needs to trim all the logs, so we don't set the cluster name on the index pattern
									Names:      []string{"tigera_secure_ee_*.*.*", "tigera_secure_ee_events.*"},
									Privileges: []string{"all"},
								}},
							},
						}},
					},
					"tigera-ee-operator": {
						Username: "tigera-ee-operator-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-operator-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor", "manage_index_templates", "manage_ilm", "read_ilm"},
								Indices: []elasticsearch.RoleIndex{{
									Names:      []string{"tigera_secure_ee_*"},
									Privileges: []string{"all"},
								}},
							},
						}},
					},
					"tigera-ee-elasticsearch-metrics": {
						Username: "tigera-ee-elasticsearch-metrics-secure",
						Roles: []elasticsearch.Role{{
							Name: "tigera-ee-elasticsearch-metrics-secure",
							Definition: &elasticsearch.RoleDefinition{
								Cluster: []string{"monitor"},
								Indices: []elasticsearch.RoleIndex{{
									Names:      []string{"*"},
									Privileges: []string{"monitor"},
								}},
							},
						}},
					},
				},
				map[users.ElasticsearchUserName]elasticsearch.User{
					"tigera-fluentd": {
						Username: "tigera-fluentd",
					},
					"tigera-eks-log-forwarder": {
						Username: "tigera-eks-log-forwarder",
					},
					"tigera-ee-compliance-benchmarker": {
						Username: "tigera-ee-compliance-benchmarker",
					},
					"tigera-ee-compliance-controller": {
						Username: "tigera-ee-compliance-controller",
					},
					"tigera-ee-compliance-reporter": {
						Username: "tigera-ee-compliance-reporter",
					},
					"tigera-ee-compliance-snapshotter": {
						Username: "tigera-ee-compliance-snapshotter",
					},
					"tigera-ee-intrusion-detection": {
						Username: "tigera-ee-intrusion-detection",
					},
					"tigera-ee-installer": {
						Username: "tigera-ee-installer",
					},
					"tigera-ee-ad-job": {
						Username: "tigera-ee-ad-job",
					},
					"tigera-ee-sasha": {
						Username: "tigera-ee-sasha",
					},
					"tigera-ee-performance-hotspots": {
						Username: "tigera-ee-performance-hotspots",
					},
					"tigera-ee-compliance-server": {
						Username: "tigera-ee-compliance-server",
					},
					"tigera-ee-manager": {
						Username: "tigera-ee-manager",
					},
					"tigera-ee-curator": {
						Username: "tigera-ee-curator",
					},
					"tigera-ee-operator": {
						Username: "tigera-ee-operator",
					},
					"tigera-ee-elasticsearch-metrics": {
						Username: "tigera-ee-elasticsearch-metrics",
					},
				},
			)
		})
	})
})

func testElasticsearchUsers(privateUsers, publicUsers, expectedprivateUsers, expectedpublicUsers map[users.ElasticsearchUserName]elasticsearch.User) {
	Expect(len(privateUsers)).Should(Equal(len(expectedprivateUsers)))
	Expect(len(publicUsers)).Should(Equal(len(expectedpublicUsers)))
	for expectedName, expectedUser := range expectedprivateUsers {
		esUser, exists := privateUsers[expectedName]
		Expect(exists).Should(BeTrue())
		Expect(esUser.Username).Should(Equal(expectedUser.Username))

		Expect(len(esUser.Roles)).Should(Equal(len(expectedUser.Roles)))

		for _, expectedRole := range expectedUser.Roles {
			for _, role := range esUser.Roles {
				if expectedRole.Name == role.Name {
					log.Printf("%s, %s", expectedRole.Name, role.Name)
					Expect(expectedRole.Definition).Should(Equal(role.Definition))
				}
			}
		}
	}
	for expectedName, expectedUser := range publicUsers {
		esUser, exists := publicUsers[expectedName]
		Expect(exists).Should(BeTrue())
		Expect(esUser.Username).Should(Equal(expectedUser.Username))

		Expect(len(esUser.Roles)).Should(Equal(len(expectedUser.Roles)))

		for _, expectedRole := range expectedUser.Roles {
			for _, role := range esUser.Roles {
				if expectedRole.Name == role.Name {
					Expect(expectedRole.Definition).Should(Equal(role.Definition))
				}
			}
		}
	}
}
