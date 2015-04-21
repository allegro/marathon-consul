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

	forwarder := Forwarder{putter, 3, opts, false}

	// single Put
	errs := forwarder.MultiPut([]*api.KVPair{good})
	assert.Equal(t, 1, len(errs))
	assert.Nil(t, errs[0])

	// multiple puts, preserving order
	errs = forwarder.MultiPut([]*api.KVPair{good, good, good, bad, good, good, good})
	assert.Equal(t, 7, len(errs))
	assert.Equal(t, []error{nil, nil, nil}, errs[:3])
	assert.Equal(t, badErr, errs[3])
	assert.Equal(t, []error{nil, nil, nil}, errs[4:])

	putter.AssertExpectations(t)
}

func TestForwardApps(t *testing.T) {
	t.Parallel()

	opts := &api.WriteOptions{}
	putter := &mocks.Putter{}
	for _, kv := range testApp.KVs() {
		putter.On("Put", kv, opts).Return(nil, nil).Twice()
	}

	forwarder := Forwarder{putter, 3, opts, false}
	errors := forwarder.ForwardApps([]*App{&testApp, &testApp})
	for _, err := range errors {
		assert.Nil(t, err)
	}

	putter.AssertExpectations(t)
}
