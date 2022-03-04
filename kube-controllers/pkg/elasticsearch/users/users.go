// Copyright (c) 2019-2021 Tigera, Inc. All rights reserved.

// package users contains the current elasticsearch users that can be used in a k8s cluster

package users

import (
	"regexp"

	"github.com/projectcalico/calico/kube-controllers/pkg/elasticsearch"
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
	ElasticsearchUserNameADJob                 ElasticsearchUserName = "tigera-ee-ad-job"
	ElasticsearchUserNameSasha                 ElasticsearchUserName = "tigera-ee-sasha"
	ElasticsearchUserNamePerformanceHotspots   ElasticsearchUserName = "tigera-ee-performance-hotspots"
	ElasticsearchUserNameInstaller             ElasticsearchUserName = "tigera-ee-installer"
	ElasticsearchUserNameManager               ElasticsearchUserName = "tigera-ee-manager"
	ElasticsearchUserNameCurator               ElasticsearchUserName = "tigera-ee-curator"
	ElasticsearchUserNameOperator              ElasticsearchUserName = "tigera-ee-operator"
	ElasticsearchUserNameElasticsearchMetrics  ElasticsearchUserName = "tigera-ee-elasticsearch-metrics"

	// This suffix is used to maintain a 1:1 mapping of public users that can be safely propagated to the managed cluster and
	// private users that will be swapped into the request at the ES gateway in the management cluster by stripping this suffix.
	ElasticsearchSecureUserSuffix = "secure"
)

// ElasticsearchUsers returns two maps of ElasticsearchUserNames as keys and elasticsearch.Users as values. The first map contains
// private Elasticsearch users with permissions and the second contains public credentials to be given to components and swapped
// out by the ES Gateway. The clusterName is used to format the username / role names for the elasticsearch.User (format is <name>-<clusterName>).
// If management is true, the return map will contain the elasticsearch users needed for a management cluster, and the usernames and
// role names will not be formatted with the clusterName.
//
// Note that the clusterName parameter is also used to format the index names, allowing the user to have access to only
// specific cluster indices
func ElasticsearchUsers(clusterName string, management bool) (map[ElasticsearchUserName]elasticsearch.User, map[ElasticsearchUserName]elasticsearch.User) {
	privateUsers := map[ElasticsearchUserName]elasticsearch.User{
		ElasticsearchUserNameFluentd: {
			Username: formatName(ElasticsearchUserNameFluentd, clusterName, management, true),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameFluentd, clusterName, management, true),
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
			Username: formatName(ElasticsearchUserNameEKSLogForwarder, clusterName, management, true),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameEKSLogForwarder, clusterName, management, true),
				Definition: &elasticsearch.RoleDefinition{
					Cluster: []string{"monitor", "manage_index_templates", "manage_ilm"},
					Indices: []elasticsearch.RoleIndex{{
						Names:      []string{indexPattern("tigera_secure_ee_audit_kube", clusterName, ".*")},
						Privileges: []string{"create_index", "read", "write", "manage"},
					}},
				},
			}},
		},
		ElasticsearchUserNameComplianceBenchmarker: {
			Username: formatName(ElasticsearchUserNameComplianceBenchmarker, clusterName, management, true),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameComplianceBenchmarker, clusterName, management, true),
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
			Username: formatName(ElasticsearchUserNameComplianceController, clusterName, management, true),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameComplianceController, clusterName, management, true),
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
			Username: formatName(ElasticsearchUserNameComplianceReporter, clusterName, management, true),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameComplianceReporter, clusterName, management, true),
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
			Username: formatName(ElasticsearchUserNameComplianceSnapshotter, clusterName, management, true),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameComplianceSnapshotter, clusterName, management, true),
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
			Username: formatName(ElasticsearchUserNameIntrusionDetection, clusterName, management, true),
			Roles: []elasticsearch.Role{
				{
					Name: formatName(ElasticsearchUserNameIntrusionDetection, clusterName, management, true),
					Definition: &elasticsearch.RoleDefinition{
						Cluster: []string{"monitor", "manage_index_templates"},
						Indices: buildElasticsearchIntrusionDetectionUserRoleIndex(clusterName, management),
					},
				},
				{
					Name: "watcher_admin",
				},
			},
		},
		ElasticsearchUserNameADJob: {
			Username: formatName(ElasticsearchUserNameADJob, clusterName, management, true),
			Roles: []elasticsearch.Role{
				{
					Name: formatName(ElasticsearchUserNameADJob, clusterName, management, true),
					Definition: &elasticsearch.RoleDefinition{
						Cluster: []string{"monitor", "manage_index_templates"},
						Indices: buildElasticsearchADJobUserRoleIndex(clusterName, management),
					},
				},
			},
		},
		ElasticsearchUserNamePerformanceHotspots: {
			Username: formatName(ElasticsearchUserNamePerformanceHotspots, clusterName, management, true),
			Roles: []elasticsearch.Role{
				{
					Name: formatName(ElasticsearchUserNamePerformanceHotspots, clusterName, management, true),
					Definition: &elasticsearch.RoleDefinition{
						Cluster: []string{"monitor", "manage_index_templates"},
						Indices: []elasticsearch.RoleIndex{
							{
								Names:      []string{indexPattern("tigera_secure_ee_flows", clusterName, ".*")},
								Privileges: []string{"read"},
							},
							{
								Names:      []string{indexPattern("tigera_secure_ee_dns", clusterName, ".*")},
								Privileges: []string{"read"},
							},
							{
								Names:      []string{indexPattern("tigera_secure_ee_l7", clusterName, ".*")},
								Privileges: []string{"read"},
							},
							{
								Names:      []string{indexPattern("tigera_secure_ee_events", clusterName, ".*")},
								Privileges: []string{"read", "write"},
							},
						},
					},
				},
			},
		},
	}
	publicUsers := map[ElasticsearchUserName]elasticsearch.User{
		ElasticsearchUserNameFluentd: {
			Username: formatName(ElasticsearchUserNameFluentd, clusterName, management, false),
		},
		ElasticsearchUserNameEKSLogForwarder: {
			Username: formatName(ElasticsearchUserNameEKSLogForwarder, clusterName, management, false),
		},
		ElasticsearchUserNameComplianceBenchmarker: {
			Username: formatName(ElasticsearchUserNameComplianceBenchmarker, clusterName, management, false),
		},
		ElasticsearchUserNameComplianceController: {
			Username: formatName(ElasticsearchUserNameComplianceController, clusterName, management, false),
		},
		ElasticsearchUserNameComplianceReporter: {
			Username: formatName(ElasticsearchUserNameComplianceReporter, clusterName, management, false),
		},
		ElasticsearchUserNameComplianceSnapshotter: {
			Username: formatName(ElasticsearchUserNameComplianceSnapshotter, clusterName, management, false),
		},
		ElasticsearchUserNameIntrusionDetection: {
			Username: formatName(ElasticsearchUserNameIntrusionDetection, clusterName, management, false),
		},
		ElasticsearchUserNameADJob: {
			Username: formatName(ElasticsearchUserNameADJob, clusterName, management, false),
		},
		ElasticsearchUserNamePerformanceHotspots: {
			Username: formatName(ElasticsearchUserNamePerformanceHotspots, clusterName, management, false),
		},
	}

	if management {
		privateManagementUsers, publicManagementUsers := managementOnlyElasticsearchUsers(clusterName)
		for k, v := range privateManagementUsers {
			privateUsers[k] = v
		}
		for k, v := range publicManagementUsers {
			publicUsers[k] = v
		}
	}
	return privateUsers, publicUsers
}

func buildElasticsearchSashaUserRoleIndex(clusterName string, isManagement bool) []elasticsearch.RoleIndex {
	clusterPattern := clusterName
	if isManagement {
		clusterPattern = "*"
	}
	return []elasticsearch.RoleIndex{
		{
			Names:      []string{indexPattern("tigera_secure_ee_runtime", clusterPattern, ".*")},
			Privileges: []string{"read"},
		},
		{
			Names:      []string{indexPattern("tigera_secure_ee_events", clusterPattern, ".*")},
			Privileges: []string{"read", "write"},
		},
	}
}

func buildElasticsearchIntrusionDetectionUserRoleIndex(clusterName string, isManagement bool) []elasticsearch.RoleIndex {
	allPrivileges := elasticsearch.RoleIndex{
		Names: []string{
			indexPattern(".tigera.ipset", clusterName, ""),
			indexPattern(".tigera.domainnameset", clusterName, ""),
			indexPattern(".tigera.forwarderconfig", clusterName, ""),
		},
		Privileges: []string{"all"},
	}

	readPrivileges := []elasticsearch.RoleIndex{
		{
			Names:      []string{indexPattern("tigera_secure_ee_*", clusterName, ".*")},
			Privileges: []string{"read"},
		},
	}
	// When configuring a management cluster we need to provide permissions for the indices across all clusters
	// (used by the IDS alert forwarding and GlobalAlerts feature).
	// Otherwise, we only need permissions to the indices specific for that individual cluster.
	if isManagement {
		allPrivileges.Names = append(allPrivileges.Names, indexPattern("tigera_secure_ee_events", "*", ""))

		datasetReadPrivileges := []elasticsearch.RoleIndex{
			{
				Names:      []string{indexPattern("tigera_secure_ee_flows", "*", ".*")},
				Privileges: []string{"read"},
			},
			{
				Names:      []string{indexPattern("tigera_secure_ee_audit_*", "*", ".*")},
				Privileges: []string{"read"},
			},
			{
				Names:      []string{indexPattern("tigera_secure_ee_dns", "*", ".*")},
				Privileges: []string{"read"},
			}}
		readPrivileges = append(readPrivileges, datasetReadPrivileges...)
	} else {
		allPrivileges.Names = append(allPrivileges.Names, indexPattern("tigera_secure_ee_events", clusterName, "*"))
	}
	return append(readPrivileges, allPrivileges)
}

// buildElasticsearchADJobUserRoleIndex creates the ES Indexes roles for AnomalyDetection Jobs Pods. The
// management cluster will have acess to all tigera_secure_ee indices for all clusters since ADJobs
// solely run on the management cluster,
func buildElasticsearchADJobUserRoleIndex(clusterName string, isManagement bool) []elasticsearch.RoleIndex {
	esADJobIndexPatternClusterName := clusterName
	if isManagement {
		esADJobIndexPatternClusterName = "*"
	}

	return []elasticsearch.RoleIndex{
		{
			Names:      []string{indexPattern("tigera_secure_ee_flows", esADJobIndexPatternClusterName, ".*")},
			Privileges: []string{"read"},
		},
		{
			Names:      []string{indexPattern("tigera_secure_ee_dns", esADJobIndexPatternClusterName, ".*")},
			Privileges: []string{"read"},
		},
		{
			Names:      []string{indexPattern("tigera_secure_ee_l7", esADJobIndexPatternClusterName, ".*")},
			Privileges: []string{"read"},
		},
		{
			Names:      []string{indexPattern("tigera_secure_ee_runtime", esADJobIndexPatternClusterName, ".*")},
			Privileges: []string{"read"},
		},
		{
			Names:      []string{indexPattern("tigera_secure_ee_events", esADJobIndexPatternClusterName, ".*")},
			Privileges: []string{"read", "write"},
		},
	}
}

func buildManagedUserPattern() []*regexp.Regexp {
	var usersPattern []*regexp.Regexp
	// Ignore public users here since they are not created in Elasticsearch.
	users, _ := ElasticsearchUsers("(.*)", false)
	for _, user := range users {
		usersPattern = append(usersPattern, regexp.MustCompile(user.Username))
	}

	return usersPattern
}

func managementOnlyElasticsearchUsers(clusterName string) (map[ElasticsearchUserName]elasticsearch.User, map[ElasticsearchUserName]elasticsearch.User) {
	privateUsers := map[ElasticsearchUserName]elasticsearch.User{
		ElasticsearchUserNameComplianceServer: {
			Username: formatName(ElasticsearchUserNameComplianceServer, clusterName, true, true),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameComplianceServer, clusterName, true, true),
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
			Username: formatName(ElasticsearchUserNameManager, clusterName, true, true),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameManager, clusterName, true, true),
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
			Username: formatName(ElasticsearchUserNameCurator, clusterName, true, true),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameCurator, clusterName, true, true),
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
			Username: formatName(ElasticsearchUserNameOperator, clusterName, true, true),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameOperator, clusterName, true, true),
				Definition: &elasticsearch.RoleDefinition{
					Cluster: []string{"monitor", "manage_index_templates", "manage_ilm", "read_ilm"},
					Indices: []elasticsearch.RoleIndex{{
						Names:      []string{"tigera_secure_ee_*"},
						Privileges: []string{"all"},
					}},
				},
			}},
		},
		ElasticsearchUserNameInstaller: {
			Username: formatName(ElasticsearchUserNameInstaller, clusterName, true, true),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameInstaller, clusterName, true, true),
				Definition: &elasticsearch.RoleDefinition{
					Cluster: []string{"manage_watcher", "manage"},
					Indices: []elasticsearch.RoleIndex{
						{
							Names:      []string{indexPattern("tigera_secure_ee_*", clusterName, ".*"), indexPattern("tigera_secure_ee_events", clusterName, ".*")},
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
		ElasticsearchUserNameElasticsearchMetrics: {
			Username: formatName(ElasticsearchUserNameElasticsearchMetrics, clusterName, true, true),
			Roles: []elasticsearch.Role{{
				Name: formatName(ElasticsearchUserNameElasticsearchMetrics, clusterName, true, true),
				Definition: &elasticsearch.RoleDefinition{
					Cluster: []string{"monitor"},
					Indices: []elasticsearch.RoleIndex{{
						Names:      []string{"*"},
						Privileges: []string{"monitor"},
					}},
				},
			}},
		},
		ElasticsearchUserNameSasha: {
			Username: formatName(ElasticsearchUserNameSasha, clusterName, true, true),
			Roles: []elasticsearch.Role{
				{
					Name: formatName(ElasticsearchUserNameSasha, clusterName, true, true),
					Definition: &elasticsearch.RoleDefinition{
						Cluster: []string{"monitor", "manage_index_templates"},
						Indices: buildElasticsearchSashaUserRoleIndex(clusterName, true),
					},
				},
			},
		},
	}
	publicUsers := map[ElasticsearchUserName]elasticsearch.User{
		ElasticsearchUserNameComplianceServer: {
			Username: formatName(ElasticsearchUserNameComplianceServer, clusterName, true, false),
		},
		ElasticsearchUserNameManager: {
			Username: formatName(ElasticsearchUserNameManager, clusterName, true, false),
		},
		ElasticsearchUserNameCurator: {
			Username: formatName(ElasticsearchUserNameCurator, clusterName, true, false),
		},
		ElasticsearchUserNameOperator: {
			Username: formatName(ElasticsearchUserNameOperator, clusterName, true, false),
		},
		ElasticsearchUserNameInstaller: {
			Username: formatName(ElasticsearchUserNameInstaller, clusterName, true, false),
		},
		ElasticsearchUserNameElasticsearchMetrics: {
			Username: formatName(ElasticsearchUserNameElasticsearchMetrics, clusterName, true, false),
		},
		ElasticsearchUserNameSasha: {
			Username: formatName(ElasticsearchUserNameSasha, clusterName, true, false),
		},
	}
	return privateUsers, publicUsers
}
