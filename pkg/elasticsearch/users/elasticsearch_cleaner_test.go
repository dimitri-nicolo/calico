// Copyright (c) 2020-2021 Tigera, Inc. All rights reserved.

package users_test

import (
	"sort"

	es "github.com/projectcalico/kube-controllers/pkg/elasticsearch"
	"github.com/projectcalico/kube-controllers/pkg/elasticsearch/users"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/mock"
)

type MockClient struct {
	userToDelete  []es.User
	rolesToDelete []es.Role
	mock.Mock
}

func (o *MockClient) CreateRoles(roles ...es.Role) error {
	args := o.Called(roles)
	return args.Error(0)
}

func (o *MockClient) CreateRoleMapping(roleMapping es.RoleMapping) error {
	args := o.Called(roleMapping)
	return args.Error(0)
}

func (o *MockClient) GetRoleMappings() ([]es.RoleMapping, error) {
	args := o.Called()
	return args.Get(0).([]es.RoleMapping), args.Error(1)
}

func (o *MockClient) DeleteRoleMapping(name string) (bool, error) {
	args := o.Called(name)
	return args.Bool(0), args.Error(1)
}

func (o *MockClient) GetUsers() ([]es.User, error) {
	args := o.Called()
	return args.Get(0).([]es.User), args.Error(1)
}

func (o *MockClient) SetUserPassword(user es.User) error {
	args := o.Called(user)
	return args.Error(0)
}

func (o *MockClient) UserExists(username string) (bool, error) {
	args := o.Called(username)
	return args.Get(0).(bool), args.Error(1)
}

func (o *MockClient) UpdateUser(user es.User) error {
	args := o.Called(user)
	return args.Error(0)
}

func (o *MockClient) DeleteUser(user es.User) error {
	args := o.Called(user)
	o.userToDelete = append(o.userToDelete, user)
	return args.Error(0)
}

func (o *MockClient) CreateUser(user es.User) error {
	args := o.Called(user)
	return args.Error(0)
}

func (o *MockClient) DeleteRole(role es.Role) error {
	args := o.Called(role)
	o.rolesToDelete = append(o.rolesToDelete, role)
	return args.Error(0)
}

var _ = Describe("ElasticSearchCleanUp", func() {

	Context("When a managed cluster is removed", func() {
		It("Should delete managed elastic users in the management cluster", func() {
			var esClient = MockClient{}
			esClient.On("DeleteUser", mock.Anything).Return(nil)
			esClient.On("DeleteRole", mock.Anything).Return(nil)

			cleaner := users.NewEsCleaner(&esClient)

			cleaner.DeleteResidueUsers("anyCluster")

			assertIssuedDeleteRequests(&esClient,
				[]string{
					"tigera-ee-compliance-benchmarker-anyCluster",
					"tigera-ee-compliance-controller-anyCluster",
					"tigera-ee-compliance-reporter-anyCluster",
					"tigera-ee-compliance-snapshotter-anyCluster",
					"tigera-ee-ad-job-anyCluster",
					"tigera-ee-intrusion-detection-anyCluster",
					"tigera-eks-log-forwarder-anyCluster",
					"tigera-fluentd-anyCluster",
				}, []string{
					"tigera-ee-compliance-benchmarker-anyCluster",
					"tigera-ee-compliance-controller-anyCluster",
					"tigera-ee-compliance-reporter-anyCluster",
					"tigera-ee-compliance-snapshotter-anyCluster",
					"tigera-ee-ad-job-anyCluster",
					"tigera-ee-intrusion-detection-anyCluster",
					"tigera-eks-log-forwarder-anyCluster",
					"tigera-fluentd-anyCluster",
				})
		})
	})

	expectedValuesOldCluster := []string{
		"tigera-ee-compliance-benchmarker-old-cluster",
		"tigera-ee-compliance-controller-old-cluster",
		"tigera-ee-compliance-reporter-old-cluster",
		"tigera-ee-compliance-snapshotter-old-cluster",
		"tigera-ee-ad-job-old-cluster",
		"tigera-ee-intrusion-detection-old-cluster",
		"tigera-eks-log-forwarder-old-cluster",
		"tigera-fluentd-old-cluster",
	}

	DescribeTable("DeleteUserAtStartUp",
		func(managedClusters map[string]bool, esUsers []es.User, expectedUserNames []string, expectedRoleNames []string) {

			var err error
			var esClient = MockClient{}
			esClient.On("GetUsers").Return(esUsers, nil)
			esClient.On("DeleteUser", mock.Anything).Return(nil)
			esClient.On("DeleteRole", mock.Anything).Return(nil)

			cleaner := users.NewEsCleaner(&esClient)

			err = cleaner.DeleteAllResidueUsers(managedClusters)
			Expect(err).NotTo(HaveOccurred())

			assertIssuedDeleteRequests(&esClient, expectedUserNames, expectedRoleNames)
		},
		Entry("Delete old users and roles",
			map[string]bool{"new-cluster": true},
			[]es.User{
				{
					Username: "tigera-ee-ad-job-old-cluster",
					Roles:    roles("tigera-ee-ad-job-role-old-cluster"),
				},
				{
					Username: "tigera-ee-ad-job",
					Roles:    roles("tigera-ee-ad-job-role"),
				},
			},
			expectedValuesOldCluster,
			expectedValuesOldCluster),
		Entry("Do not delete new users and roles",
			map[string]bool{"new-cluster": true},
			[]es.User{
				{
					Username: "tigera-ee-ad-job-new-cluster",
					Roles:    roles("tigera-ee-ad-job-role-new-cluster"),
				},
				{
					Username: "tigera-ee-ad-job",
					Roles:    roles("tigera-ee-ad-job-role"),
				},
			},
			nil,
			nil),
		Entry("Do not delete when es returns 0 users",
			map[string]bool{"new-cluster": true},
			[]es.User{},
			nil,
			nil),
		Entry("Delete users for old-clusters when k8s api returns zero managed clusters",
			map[string]bool{},
			[]es.User{
				{
					Username: "tigera-ee-ad-job-old-cluster",
					Roles:    roles("tigera-ee-ad-job-role-old-cluster"),
				},
				{
					Username: "tigera-ee-ad-job",
					Roles:    roles("tigera-ee-ad-job-role"),
				},
			},
			expectedValuesOldCluster,
			expectedValuesOldCluster),
		Entry("DO not delete non-tigera users and roles",
			map[string]bool{"any-cluster": true},
			[]es.User{
				{
					Username: "admin",
					Roles:    roles("admin"),
				},
			},
			nil,
			nil),
		Entry("Do not delete non-tigera roles",
			map[string]bool{"new-cluster": true},
			[]es.User{
				{
					Username: "tigera-ee-ad-job-old-cluster",
					Roles:    roles("watcher_admin"),
				},
				{
					Username: "tigera-ee-ad-job",
					Roles:    roles("watcher_admin"),
				},
			},
			expectedValuesOldCluster,
			expectedValuesOldCluster),
	)
})

func roles(name string) []es.Role {
	return []es.Role{
		{
			Name: name,
		},
	}
}

func assertIssuedDeleteRequests(esClient *MockClient, expectedUserNames, expectedRoleNames []string) {
	var deletedUsers []string
	var deletedRoles []string
	for _, user := range esClient.userToDelete {
		deletedUsers = append(deletedUsers, user.Username)
	}
	for _, role := range esClient.rolesToDelete {
		deletedRoles = append(deletedRoles, role.Name)
	}

	sort.Strings(deletedUsers)
	sort.Strings(deletedRoles)
	sort.Strings(expectedUserNames)
	sort.Strings(expectedRoleNames)

	Expect(deletedUsers).To(Equal(expectedUserNames))
	Expect(deletedRoles).To(Equal(expectedRoleNames))
}
