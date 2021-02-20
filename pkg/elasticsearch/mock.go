// Copyright (c) 2019-2020 Tigera, Inc. All rights reserved.

package elasticsearch

import "github.com/stretchr/testify/mock"

type MockClient struct {
	mock.Mock
}

func NewMockClient() *MockClient {
	return &MockClient{}
}

func (o *MockClient) CreateRoles(roles ...Role) error {
	args := o.Called(roles)
	return args.Error(0)
}

func (o *MockClient) CreateRoleMapping(roleMapping RoleMapping) error {
	args := o.Called(roleMapping)
	return args.Error(0)
}

func (o *MockClient) GetRoleMappings() ([]RoleMapping, error) {
	args := o.Called()
	return args.Get(0).([]RoleMapping), args.Error(1)
}

func (o *MockClient) DeleteRoleMapping(name string) (bool, error) {
	args := o.Called(name)
	return args.Bool(0), args.Error(1)
}

func (o *MockClient) GetUsers() ([]User, error) {
	args := o.Called()
	return args.Get(0).([]User), args.Error(1)
}

func (o *MockClient) UserExists(username string) (bool, error) {
	args := o.Called(username)
	return args.Get(0).(bool), args.Error(1)
}

func (o *MockClient) SetUserPassword(user User) error {
	args := o.Called(user)
	return args.Error(0)
}

func (o *MockClient) UpdateUser(user User) error {
	args := o.Called(user)
	return args.Error(0)
}

func (o *MockClient) DeleteUser(user User) error {
	args := o.Called(user)
	return args.Error(0)
}

func (o *MockClient) CreateUser(user User) error {
	args := o.Called(user)
	return args.Error(0)
}

func (o *MockClient) DeleteRole(role Role) error {
	args := o.Called(role)
	return args.Error(0)
}
