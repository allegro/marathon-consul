package mocks

import "github.com/stretchr/testify/mock"

import "github.com/hashicorp/consul/api"

type PutDeleter struct {
	mock.Mock
}

func (m PutDeleter) Put(kv *api.KVPair) (*api.WriteMeta, error) {
	ret := m.Called(kv)

	var writeMeta *api.WriteMeta
	if ret.Get(0) != nil {
		writeMeta = ret.Get(0).(*api.WriteMeta)
	}
	err := ret.Error(1)

	return writeMeta, err
}

func (m PutDeleter) Delete(key string) (*api.WriteMeta, error) {
	ret := m.Called(key)

	var writeMeta *api.WriteMeta
	if ret.Get(0) != nil {
		writeMeta = ret.Get(0).(*api.WriteMeta)
	}
	err := ret.Error(1)

	return writeMeta, err
}
