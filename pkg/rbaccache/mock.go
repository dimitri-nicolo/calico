// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package rbaccache

import (
	"github.com/stretchr/testify/mock"
	rbacv1 "k8s.io/api/rbac/v1"
)

type MockClusterRoleCache struct {
	mock.Mock
}

func NewMockClusterRoleCache() *MockClusterRoleCache {
	return &MockClusterRoleCache{}
}

func (m *MockClusterRoleCache) AddClusterRole(clusterRole *rbacv1.ClusterRole) bool {
	args := m.Called(clusterRole)
	return args.Bool(0)
}

func (m *MockClusterRoleCache) AddClusterRoleBinding(clusterRoleBinding *rbacv1.ClusterRoleBinding) bool {
	args := m.Called(clusterRoleBinding)
	return args.Bool(0)
}

func (m *MockClusterRoleCache) RemoveClusterRole(clusterRoleName string) bool {
	args := m.Called(clusterRoleName)
	return args.Bool(0)
}

func (m *MockClusterRoleCache) RemoveClusterRoleBinding(clusterRoleBindingName string) bool {
	args := m.Called(clusterRoleBindingName)
	return args.Bool(0)
}

func (m *MockClusterRoleCache) ClusterRoleSubjects(clusterRoleName string, subjectType string) []rbacv1.Subject {
	args := m.Called(clusterRoleName, subjectType)
	return args[0].([]rbacv1.Subject)
}

func (m *MockClusterRoleCache) ClusterRoleRules(clusterRoleName string) []rbacv1.PolicyRule {
	args := m.Called(clusterRoleName)
	return args[0].([]rbacv1.PolicyRule)
}

func (m *MockClusterRoleCache) ClusterRoleNameForBinding(clusterRoleBindingName string) string {
	args := m.Called(clusterRoleBindingName)
	return args.String(0)
}

func (m *MockClusterRoleCache) ClusterRoleNamesWithBindings() []string {
	args := m.Called()
	return args[0].([]string)
}

func (m *MockClusterRoleCache) ClusterRoleNamesForSubjectName(userOrGroup string) []string {
	args := m.Called(userOrGroup)
	return args[0].([]string)
}

func (m *MockClusterRoleCache) ClusterRoleBindingsForClusterRole(clusterRoleName string) []string {
	args := m.Called(clusterRoleName)
	return args[0].([]string)
}

func (m *MockClusterRoleCache) SubjectNamesForBinding(clusterRoleBindingName string) []string {
	args := m.Called(clusterRoleBindingName)
	return args[0].([]string)
}
