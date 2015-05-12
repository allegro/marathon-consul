package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/CiscoCloud/marathon-consul/mocks"
	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequest("GET", "http://example.com/health", nil)
	assert.Nil(t, err)

	recorder := httptest.NewRecorder()
	HealthHandler(recorder, req)

	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "OK\n", recorder.Body.String())
}

func TestForwardHandler(t *testing.T) {
	t.Parallel()

	// create a forwarder
	opts := &api.WriteOptions{}
	putter := &mocks.Putter{}
	errKV := errors.New("test error")
	forwarder := Forwarder{putter, 3, opts, false}
	handler := ForwardHandler{forwarder}

	body, err := json.Marshal(APIPostEvent{"api_post_event", testApp})
	assert.Nil(t, err)

	// test no JSON
	req, err := http.NewRequest("POST", "http://example.com/event", bytes.NewReader([]byte{}))
	assert.Nil(t, err)

	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	assert.Equal(t, 500, recorder.Code)
	assert.Equal(t, "could not read request body\n", recorder.Body.String())

	// test an unsupported event
	req, err = http.NewRequest("POST", "http://example.com/event", bytes.NewReader([]byte(`{"eventType":"bad_event"}`)))
	assert.Nil(t, err)

	recorder = httptest.NewRecorder()
	handler.Handle(recorder, req)

	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "this endpoint only accepts api_post_event and deployment_info\n", recorder.Body.String())

	// test a good update
	for _, kv := range testApp.KVs() {
		putter.On("Put", kv, opts).Return(nil, nil).Once()
	}

	req, err = http.NewRequest("POST", "http://example.com/event", bytes.NewReader(body))
	assert.Nil(t, err)

	recorder = httptest.NewRecorder()
	handler.Handle(recorder, req)

	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "OK\n", recorder.Body.String())
	// putter.AssertExpectations(t)

	// test a bad update
	for i, kv := range testApp.KVs() {
		if i == 0 {
			putter.On("Put", kv, opts).Return(nil, errKV).Once()
		} else {
			putter.On("Put", kv, opts).Return(nil, nil).Once()
		}
	}

	req, err = http.NewRequest("POST", "http://example.com/event", bytes.NewReader(body))
	assert.Nil(t, err)

	recorder = httptest.NewRecorder()
	handler.Handle(recorder, req)

	assert.Equal(t, 500, recorder.Code)
	assert.Equal(t, "test error\n", recorder.Body.String())
	// putter.AssertExpectations(t)
}
