package main

import (
	"encoding/json"
	"errors"
	"github.com/CiscoCloud/marathon-consul/mocks"
	"github.com/CiscoCloud/marathon-consul/tasks"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var testTask = &tasks.Task{
	Timestamp:  "2014-03-01T23:29:30.158Z",
	SlaveID:    "20140909-054127-177048842-5050-1494-0",
	TaskID:     "my-app_0-1396592784349",
	TaskStatus: "TASK_RUNNING",
	AppID:      "/my-app",
	Host:       "slave-1234.acme.org",
	Ports:      []int{31372},
	Version:    "2014-04-04T06:26:23.051Z",
}

func TestHealthHandler(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequest("GET", "http://example.com/health", nil)
	assert.Nil(t, err)

	recorder := httptest.NewRecorder()
	HealthHandler(recorder, req)

	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "OK\n", recorder.Body.String())
}

func TestForwardHandlerHandleAppEvent(t *testing.T) {
	t.Parallel()

	// create a handler
	kv := &mocks.PutDeleter{}
	errKV := errors.New("test error")
	handler := ForwardHandler{kv, false, false}

	body, err := json.Marshal(APIPostEvent{"api_post_event", testApp})
	assert.Nil(t, err)

	// test a good update
	kv.On("Put", testApp.KV()).Return(nil, nil).Once()
	recorder := httptest.NewRecorder()
	handler.HandleAppEvent(recorder, body)

	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "OK\n", recorder.Body.String())

	// test a bad update
	kv.On("Put", testApp.KV()).Return(nil, errKV).Once()
	recorder = httptest.NewRecorder()
	handler.HandleAppEvent(recorder, body)

	assert.Equal(t, 500, recorder.Code)
	assert.Equal(t, "test error\n", recorder.Body.String())
}

func TestForwardHandlerHandleTerminationEvent(t *testing.T) {
	t.Parallel()

	// create a handler
	kv := &mocks.PutDeleter{}
	errKV := errors.New("test error")
	handler := ForwardHandler{kv, false, false}

	body, err := json.Marshal(AppTerminatedEvent{
		Type:  "app_terminated_event",
		AppID: testApp.ID,
	})
	assert.Nil(t, err)

	// test a good update
	kv.On("Delete", testApp.Key()).Return(nil, nil).Once()
	recorder := httptest.NewRecorder()
	handler.HandleTerminationEvent(recorder, body)

	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "OK\n", recorder.Body.String())

	// test a bad update
	kv.On("Delete", testApp.Key()).Return(nil, errKV).Once()
	recorder = httptest.NewRecorder()
	handler.HandleTerminationEvent(recorder, body)

	assert.Equal(t, 500, recorder.Code)
	assert.Equal(t, "test error\n", recorder.Body.String())
}

func tempTaskBody(status string) []byte {
	body, _ := json.Marshal(testTask)
	return []byte(strings.Replace(
		string(body),
		testTask.TaskStatus,
		status,
		1,
	))
}

func TestForwardHandlerHandleStatusEvent(t *testing.T) {
	t.Parallel()

	// create a handler
	kv := &mocks.PutDeleter{}
	errKV := errors.New("test error")
	handler := ForwardHandler{kv, false, false}

	// deletes
	for _, status := range []string{"TASK_FINISHED", "TASK_FAILED", "TASK_KILLED", "TASK_LOST"} {
		tempBody := tempTaskBody(status)
		// good update
		kv.On("Delete", testTask.Key()).Return(nil, nil).Once()
		recorder := httptest.NewRecorder()
		handler.HandleStatusEvent(recorder, tempBody)
		assert.Equal(t, 200, recorder.Code)
		assert.Equal(t, "OK\n", recorder.Body.String())

		// bad update
		kv.On("Delete", testTask.Key()).Return(nil, errKV).Once()
		recorder = httptest.NewRecorder()
		handler.HandleStatusEvent(recorder, tempBody)
		assert.Equal(t, 500, recorder.Code)
		assert.Equal(t, "test error\n", recorder.Body.String())
	}

	// puts
	for _, status := range []string{"TASK_STAGING", "TASK_STARTING", "TASK_RUNNING"} {
		tempBody := tempTaskBody(status)
		tempTask, _ := tasks.ParseTask(tempBody)
		// good update
		kv.On("Put", tempTask.KV()).Return(nil, nil).Once()
		recorder := httptest.NewRecorder()
		handler.HandleStatusEvent(recorder, tempBody)
		assert.Equal(t, 200, recorder.Code)
		assert.Equal(t, "OK\n", recorder.Body.String())

		// bad update
		kv.On("Put", tempTask.KV()).Return(nil, errKV).Once()
		recorder = httptest.NewRecorder()
		handler.HandleStatusEvent(recorder, tempBody)
		assert.Equal(t, 500, recorder.Code)
		assert.Equal(t, "test error\n", recorder.Body.String())
	}

	// bad status
	tempBody := tempTaskBody("TASK_BATMAN")
	recorder := httptest.NewRecorder()
	handler.HandleStatusEvent(recorder, tempBody)
	assert.Equal(t, 500, recorder.Code)
	assert.Equal(t, "unknown task status\n", recorder.Body.String())
}
