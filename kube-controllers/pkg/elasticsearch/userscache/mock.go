// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package userscache

import (
	mock "github.com/stretchr/testify/mock"
)

// MockOIDCUserCache is mock type for the MockOIDCUserCache type
type MockOIDCUserCache struct {
	mock.Mock
}

func NewMockOIDCUserCache() *MockOIDCUserCache {
	return &MockOIDCUserCache{}
}

// DeleteOIDCUser provides a mock function with given fields: subjectID
func (_m *MockOIDCUserCache) DeleteOIDCUser(subjectID string) bool {
	ret := _m.Called(subjectID)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(subjectID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Exists provides a mock function with given fields: subjectID
func (_m *MockOIDCUserCache) Exists(subjectID string) bool {
	ret := _m.Called(subjectID)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string) bool); ok {
		r0 = rf(subjectID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// SubjectIDToUserOrGroups provides a mock function with given fields: subjectID
func (_m *MockOIDCUserCache) SubjectIDToUserOrGroups(subjectID string) []string {
	ret := _m.Called(subjectID)

	var r0 []string
	if rf, ok := ret.Get(0).(func(string) []string); ok {
		r0 = rf(subjectID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// SubjectIDs provides a mock function with given fields:
func (_m *MockOIDCUserCache) SubjectIDs() []string {
	ret := _m.Called()

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// UpdateOIDCUsers provides a mock function with given fields: oidcUsers
func (_m *MockOIDCUserCache) UpdateOIDCUsers(oidcUsers map[string]OIDCUser) []string {
	ret := _m.Called(oidcUsers)

	var r0 []string
	if rf, ok := ret.Get(0).(func(map[string]OIDCUser) []string); ok {
		r0 = rf(oidcUsers)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// UserOrGroupToSubjectIDs provides a mock function with given fields: userOrGroup
func (_m *MockOIDCUserCache) UserOrGroupToSubjectIDs(userOrGroup string) []string {
	ret := _m.Called(userOrGroup)

	var r0 []string
	if rf, ok := ret.Get(0).(func(string) []string); ok {
		r0 = rf(userOrGroup)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}
