package main

import (
	"errors"
	"github.com/CiscoCloud/marathon-forwarder/mocks"
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMultiPut(t *testing.T) {
	t.Parallel()
	good := &api.KVPair{Key: "good", Value: []byte("")}
	bad := &api.KVPair{Key: "bad", Value: []byte("")}
	badErr := errors.New("test error")

	opts := &api.WriteOptions{}
	putter := &mocks.Putter{}
	putter.On("Put", good, opts).Return(nil, nil)
	putter.On("Put", bad, opts).Return(nil, badErr)

	forwarder := Forwarder{putter, 3, opts}

	// single Put
	err := forwarder.MultiPut([]*api.KVPair{good})
	assert.Equal(t, 1, len(err))
	assert.Nil(t, err[0])

	// multiple puts, preserving order
	err = forwarder.MultiPut([]*api.KVPair{good, good, good, bad, good, good, good})
	assert.Equal(t, 7, len(err))
	assert.Equal(t, []error{nil, nil, nil}, err[:3])
	assert.Equal(t, badErr, err[3])
	assert.Equal(t, []error{nil, nil, nil}, err[4:])

	putter.AssertExpectations(t)
}
