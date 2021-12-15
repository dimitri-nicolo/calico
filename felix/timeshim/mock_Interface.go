// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

// Copyright (c) 2021 Tigera, Inc. All rights reserved.

package timeshim

import (
	time "time"

	mock "github.com/stretchr/testify/mock"
)

// MockInterface is an autogenerated mock type for the Interface type
type MockInterface struct {
	mock.Mock
}

// After provides a mock function with given fields: t
func (_m *MockInterface) After(t time.Duration) <-chan time.Time {
	ret := _m.Called(t)

	var r0 <-chan time.Time
	if rf, ok := ret.Get(0).(func(time.Duration) <-chan time.Time); ok {
		r0 = rf(t)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan time.Time)
		}
	}

	return r0
}

// KTimeNanos provides a mock function with given fields:
func (_m *MockInterface) KTimeNanos() int64 {
	ret := _m.Called()

	var r0 int64
	if rf, ok := ret.Get(0).(func() int64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int64)
	}

	return r0
}

// NewTicker provides a mock function with given fields: d
func (_m *MockInterface) NewTicker(d time.Duration) Ticker {
	ret := _m.Called(d)

	var r0 Ticker
	if rf, ok := ret.Get(0).(func(time.Duration) Ticker); ok {
		r0 = rf(d)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(Ticker)
		}
	}

	return r0
}

// NewTimer provides a mock function with given fields: d
func (_m *MockInterface) NewTimer(d time.Duration) Timer {
	ret := _m.Called(d)

	var r0 Timer
	if rf, ok := ret.Get(0).(func(time.Duration) Timer); ok {
		r0 = rf(d)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(Timer)
		}
	}

	return r0
}

// Now provides a mock function with given fields:
func (_m *MockInterface) Now() time.Time {
	ret := _m.Called()

	var r0 time.Time
	if rf, ok := ret.Get(0).(func() time.Time); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	return r0
}

// Since provides a mock function with given fields: t
func (_m *MockInterface) Since(t time.Time) time.Duration {
	ret := _m.Called(t)

	var r0 time.Duration
	if rf, ok := ret.Get(0).(func(time.Time) time.Duration); ok {
		r0 = rf(t)
	} else {
		r0 = ret.Get(0).(time.Duration)
	}

	return r0
}

// Until provides a mock function with given fields: t
func (_m *MockInterface) Until(t time.Time) time.Duration {
	ret := _m.Called(t)

	var r0 time.Duration
	if rf, ok := ret.Get(0).(func(time.Time) time.Duration); ok {
		r0 = rf(t)
	} else {
		r0 = ret.Get(0).(time.Duration)
	}

	return r0
}
