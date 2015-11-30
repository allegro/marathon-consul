package tasks

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testTask = &Task{
	Timestamp:  "2014-03-01T23:29:30.158Z",
	SlaveID:    "20140909-054127-177048842-5050-1494-0",
	ID:         "my-app_0-1396592784349",
	TaskStatus: "TASK_RUNNING",
	AppID:      "/my-app",
	Host:       "slave-1234.acme.org",
	Ports:      []int{31372},
	Version:    "2014-04-04T06:26:23.051Z",
	HealthCheckResults: []HealthCheckResult{HealthCheckResult{Alive:true}},
}

func TestParseTask(t *testing.T) {
	t.Parallel()

	jsonified, err := json.Marshal(testTask)
	assert.Nil(t, err)

	service, err := ParseTask(jsonified)
	assert.Nil(t, err)

	assert.Equal(t, testTask.Timestamp, service.Timestamp)
	assert.Equal(t, testTask.SlaveID, service.SlaveID)
	assert.Equal(t, testTask.ID, service.ID)
	assert.Equal(t, testTask.TaskStatus, service.TaskStatus)
	assert.Equal(t, testTask.AppID, service.AppID)
	assert.Equal(t, testTask.Host, service.Host)
	assert.Equal(t, testTask.Ports, service.Ports)
	assert.Equal(t, testTask.Version, service.Version)
	assert.Equal(t, testTask.HealthCheckResults[0].Alive, service.HealthCheckResults[0].Alive)
}

func TestKV(t *testing.T) {
	t.Parallel()

	kv := testTask.KV()

	jsonified, err := json.Marshal(testTask)
	assert.Nil(t, err)

	assert.Equal(t, fmt.Sprintf("%s/tasks/%s", "my-app", testTask.ID), kv.Key)
	assert.Equal(t, jsonified, kv.Value)
}
