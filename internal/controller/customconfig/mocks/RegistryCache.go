// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// RegistryCache is an autogenerated mock type for the RegistryCache type
type RegistryCache struct {
	mock.Mock
}

// RegistryCacheConfigExists provides a mock function with no fields
func (_m *RegistryCache) RegistryCacheConfigExists() (bool, error) {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for RegistryCacheConfigExists")
	}

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func() (bool, error)); ok {
		return rf()
	}
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewRegistryCache creates a new instance of RegistryCache. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewRegistryCache(t interface {
	mock.TestingT
	Cleanup(func())
}) *RegistryCache {
	mock := &RegistryCache{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
