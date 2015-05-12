package main

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testService = &Service{
	Timestamp:  "2014-03-01T23:29:30.158Z",
	SlaveID:    "20140909-054127-177048842-5050-1494-0",
	TaskID:     "my-app_0-1396592784349",
	TaskStatus: "TASK_RUNNING",
	AppID:      "/my-app",
	Host:       "slave-1234.acme.org",
	Ports:      []int{31372},
	Version:    "2014-04-04T06:26:23.051Z",
}

func TestParseService(t *testing.T) {
	t.Parallel()

	jsonified, err := json.Marshal(testService)
	assert.Nil(t, err)

	service, err := ParseService(jsonified)
	assert.Nil(t, err)

	assert.Equal(t, testService.Timestamp, service.Timestamp)
	assert.Equal(t, testService.SlaveID, service.SlaveID)
	assert.Equal(t, testService.TaskID, service.TaskID)
	assert.Equal(t, testService.TaskStatus, service.TaskStatus)
	assert.Equal(t, testService.AppID, service.AppID)
	assert.Equal(t, testService.Host, service.Host)
	assert.Equal(t, testService.Ports, service.Ports)
	assert.Equal(t, testService.Version, service.Version)
}

func TestKV(t *testing.T) {
	t.Parallel()

	kv := testService.KV()

	jsonified, err := json.Marshal(testService)
	assert.Nil(t, err)

	assert.Equal(t, testService.TaskID, kv.Key)
	assert.Equal(t, jsonified, kv.Value)
}
