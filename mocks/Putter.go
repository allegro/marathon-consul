package mocks

import "github.com/stretchr/testify/mock"

import "github.com/hashicorp/consul/api"

type Putter struct {
	mock.Mock
}

func (m *Putter) Put(kv *api.KVPair, opts *api.WriteOptions) (*api.WriteMeta, error) {
	ret := m.Called(kv, opts)

	var writeMeta *api.WriteMeta
	if ret.Get(0) != nil {
		writeMeta = ret.Get(0).(*api.WriteMeta)
	}
	err := ret.Error(1)

	return writeMeta, err
}
