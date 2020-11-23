// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

// package users contains the current elasticsearch users that can be used in a k8s cluster

package users

import (
	"fmt"
	"regexp"

	"github.com/projectcalico/kube-controllers/pkg/elasticsearch"
)

type ElasticsearchUserName string

const (
	ElasticsearchUserNameFluentd               ElasticsearchUserName = "tigera-fluentd"
	ElasticsearchUserNameEKSLogForwarder       ElasticsearchUserName = "tigera-eks-log-forwarder"
	ElasticsearchUserNameComplianceBenchmarker ElasticsearchUserName = "tigera-ee-compliance-benchmarker"
	ElasticsearchUserNameComplianceController  ElasticsearchUserName = "tigera-ee-compliance-controller"
	ElasticsearchUserNameComplianceReporter    ElasticsearchUserName = "tigera-ee-compliance-reporter"
	ElasticsearchUserNameComplianceSnapshotter ElasticsearchUserName = "tigera-ee-compliance-snapshotter"
	ElasticsearchUserNameComplianceServer      ElasticsearchUserName = "tigera-ee-compliance-server"
	ElasticsearchUserNameIntrusionDetection    ElasticsearchUserName = "tigera-ee-intrusion-detection"
	ElasticsearchUserNameInstaller             ElasticsearchUserName = "tigera-ee-installer"
	ElasticsearchUserNameManager               ElasticsearchUserName = "tigera-ee-manager"
	ElasticsearchUserNameCurator               ElasticsearchUserName = "tigera-ee-curator"
	ElasticsearchUserNameOperator              ElasticsearchUserName = "tigera-ee-operator"
)

// ElasticsearchUsers returns a map of ElasticsearchUserNames as keys and elasticsearch.Users as values. The clusterName
// is used to format the username / role names for the elasticsearch.User (format is <name>-<clusterName>). If management
// is true, the return map will contain the elasticsearch users needed for a management cluster, and the usernames and
// role names will not be formatted with the clusterName.
//
// Note that the clusterName parameter is also used to format the index names, allowing the user to have access to only
// specific cluster indices
func ElasticsearchUsers(clusterName string, management bool) map[ElasticsearchUserName]elasticsearch.User {
	users := map[ElasticsearchUserName]elasticsearch.User{
		ElasticsearchUserNameFluentd: {
			Username: formatName(ElasticsearchUserNameFluentd, clusterName, management),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameFluentd, clusterName, management),
				Definition: &elasticsearch.RoleDefinition{
					Cluster: []string{"monitor", "manage_index_templates", "manage_ilm"},
					Indices: []elasticsearch.RoleIndex{{
						Names:      []string{indexPattern("tigera_secure_ee_*", clusterName, ".*")},
						Privileges: []string{"create_index", "write", "manage"},
					}},
				},
			}},
		},
		ElasticsearchUserNameEKSLogForwarder: {
			Username: formatName(ElasticsearchUserNameEKSLogForwarder, clusterName, management),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameEKSLogForwarder, clusterName, management),
				Definition: &elasticsearch.RoleDefinition{
					Cluster: []string{"monitor", "manage_index_templates"},
					Indices: []elasticsearch.RoleIndex{{
						Names:      []string{indexPattern("tigera_secure_ee_audit_kube", clusterName, ".*")},
						Privileges: []string{"create_index", "read", "write", "manage"},
					}},
				},
			}},
		},
		ElasticsearchUserNameComplianceBenchmarker: {
			Username: formatName(ElasticsearchUserNameComplianceBenchmarker, clusterName, management),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameComplianceBenchmarker, clusterName, management),
				Definition: &elasticsearch.RoleDefinition{
					Cluster: []string{"monitor", "manage_index_templates"},
					Indices: []elasticsearch.RoleIndex{{
						Names:      []string{indexPattern("tigera_secure_ee_benchmark_results", clusterName, ".*")},
						Privileges: []string{"create_index", "write", "view_index_metadata", "read", "manage"},
					}},
				},
			}},
		},
		ElasticsearchUserNameComplianceController: {
			Username: formatName(ElasticsearchUserNameComplianceController, clusterName, management),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameComplianceController, clusterName, management),
				Definition: &elasticsearch.RoleDefinition{
					Cluster: []string{"monitor", "manage_index_templates"},
					Indices: []elasticsearch.RoleIndex{{
						Names:      []string{indexPattern("tigera_secure_ee_compliance_reports", clusterName, ".*")},
						Privileges: []string{"read"},
					}},
				},
			}},
		},
		ElasticsearchUserNameComplianceReporter: {
			Username: formatName(ElasticsearchUserNameComplianceReporter, clusterName, management),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameComplianceReporter, clusterName, management),
				Definition: &elasticsearch.RoleDefinition{
					Cluster: []string{"monitor", "manage_index_templates"},
					Indices: []elasticsearch.RoleIndex{
						{
							Names:      []string{indexPattern("tigera_secure_ee_audit_*", clusterName, ".*")},
							Privileges: []string{"read"},
						},
						{
							Names:      []string{indexPattern("tigera_secure_ee_snapshots", clusterName, ".*")},
							Privileges: []string{"read"},
						},
						{
							Names:      []string{indexPattern("tigera_secure_ee_benchmark_results", clusterName, ".*")},
							Privileges: []string{"read"},
						},
						{
							Names:      []string{indexPattern("tigera_secure_ee_flows", clusterName, ".*")},
							Privileges: []string{"read"},
						},
						{
							Names:      []string{indexPattern("tigera_secure_ee_compliance_reports", clusterName, ".*")},
							Privileges: []string{"create_index", "write", "view_index_metadata", "read", "manage"},
						},
					},
				},
			}},
		},
		ElasticsearchUserNameComplianceSnapshotter: {
			Username: formatName(ElasticsearchUserNameComplianceSnapshotter, clusterName, management),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameComplianceSnapshotter, clusterName, management),
				Definition: &elasticsearch.RoleDefinition{
					Cluster: []string{"monitor", "manage_index_templates"},
					Indices: []elasticsearch.RoleIndex{{
						Names:      []string{indexPattern("tigera_secure_ee_snapshots", clusterName, ".*")},
						Privileges: []string{"create_index", "write", "view_index_metadata", "read", "manage"},
					}},
				},
			}},
		},
		ElasticsearchUserNameIntrusionDetection: {
			Username: formatName(ElasticsearchUserNameIntrusionDetection, clusterName, management),
			Roles: []elasticsearch.Role{
				{
					Name: formatName(ElasticsearchUserNameIntrusionDetection, clusterName, management),
					Definition: &elasticsearch.RoleDefinition{
						Cluster: []string{"monitor", "manage_index_templates"},
						Indices: []elasticsearch.RoleIndex{
							{
								Names:      []string{indexPattern("tigera_secure_ee_*", clusterName, ".*")},
								Privileges: []string{"read"},
							},
							{
								Names: []string{
									indexPattern(".tigera.ipset", clusterName, ""),
									indexPattern("tigera_secure_ee_events", clusterName, ""),
									indexPattern(".tigera.domainnameset", clusterName, ""),
									indexPattern(".tigera.forwarderconfig", clusterName, ""),
								},
								Privileges: []string{"all"},
							},
						},
					},
				},
				{
					// TODO look into this role, it may give managed clusters abilities we don't want them to have (SAAS-626)
					Name: "watcher_admin",
				},
			},
		},
		ElasticsearchUserNameInstaller: {
			Username: formatName(ElasticsearchUserNameInstaller, clusterName, management),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameInstaller, clusterName, management),
				Definition: &elasticsearch.RoleDefinition{
					Cluster: []string{"manage_ml", "manage_watcher", "manage"},
					Indices: []elasticsearch.RoleIndex{
						{
							Names:      []string{indexPattern("tigera_secure_ee_*", clusterName, ".*"), indexPattern("tigera_secure_ee_events", clusterName, "")},
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
	}

	if management {
		for k, v := range managementOnlyElasticsearchUsers(clusterName) {
			users[k] = v
		}
	}

	return users
}

func buildManagedUserPattern() []*regexp.Regexp {
	var usersPattern []*regexp.Regexp
	users := ElasticsearchUsers("(.*)", false)
	for _, user := range users {
		usersPattern = append(usersPattern, regexp.MustCompile(user.Username))
	}

	return usersPattern
}

func managementOnlyElasticsearchUsers(clusterName string) map[ElasticsearchUserName]elasticsearch.User {
	return map[ElasticsearchUserName]elasticsearch.User{
		ElasticsearchUserNameComplianceServer: {
			Username: formatName(ElasticsearchUserNameComplianceServer, clusterName, true),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameComplianceServer, clusterName, true),
				Definition: &elasticsearch.RoleDefinition{
					Cluster: []string{"monitor", "manage_index_templates"},
					Indices: []elasticsearch.RoleIndex{{
						// ComplianceServer needs access to all indices as it creates reports for all the clusters
						Names:      []string{indexPattern("tigera_secure_ee_compliance_reports", "*", ".*")},
						Privileges: []string{"read"},
					}},
				},
			}},
		},
		ElasticsearchUserNameManager: {
			Username: formatName(ElasticsearchUserNameManager, clusterName, true),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameManager, clusterName, true),
				Definition: &elasticsearch.RoleDefinition{
					Cluster: []string{"monitor"},
					Indices: []elasticsearch.RoleIndex{{
						// Let the manager query elasticsearch for all clusters for multicluster management.
						Names:      []string{indexPattern("tigera_secure_ee_*", "*", ".*"), indexPattern("tigera_secure_ee_events", "*", ""), ".kibana"},
						Privileges: []string{"read"},
					}},
				},
			}},
		},
		ElasticsearchUserNameCurator: {
			Username: formatName(ElasticsearchUserNameCurator, clusterName, true),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameCurator, clusterName, true),
				Definition: &elasticsearch.RoleDefinition{
					Cluster: []string{"monitor", "manage_index_templates"},
					Indices: []elasticsearch.RoleIndex{{
						// Curator needs to trim all the logs, so we don't set the cluster name on the index pattern
						Names:      []string{indexPattern("tigera_secure_ee_*", "*", ".*"), indexPattern("tigera_secure_ee_events", "*", "")},
						Privileges: []string{"all"},
					}},
				},
			}},
		},
		ElasticsearchUserNameOperator: {
			Username: formatName(ElasticsearchUserNameOperator, clusterName, true),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameOperator, clusterName, true),
				Definition: &elasticsearch.RoleDefinition{
					Cluster: []string{"monitor", "manage_index_templates", "manage_ilm", "read_ilm"},
					Indices: []elasticsearch.RoleIndex{{
						Names:      []string{"tigera_secure_ee_*"},
						Privileges: []string{"all"},
					}},
				},
			}},
		},
	}
}

func indexPattern(prefix string, cluster string, suffix string) string {
	return fmt.Sprintf("%s.%s%s", prefix, cluster, suffix)
}

func formatName(name ElasticsearchUserName, clusterName string, management bool) string {
	if management {
		return string(name)
	}
	return fmt.Sprintf("%s-%s", string(name), clusterName)
}
