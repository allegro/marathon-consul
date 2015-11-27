package tasks

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

var testHealthChange = &TaskHealthChange{
	Timestamp:  "2014-03-01T23:29:30.158Z",
	ID:         "my-app_0-1396592784349",
	Alive: true,
	TaskStatus: "TASK_RUNNING",
	AppID:      "/my-app",
	Version:    "2014-04-04T06:26:23.051Z",
}

func TestHealthChangeParseTask(t *testing.T) {
	t.Parallel()

	jsonified, err := json.Marshal(testHealthChange)
	assert.Nil(t, err)

	service, err := ParseTaskHealthChange(jsonified)
	assert.Nil(t, err)

	assert.Equal(t, testHealthChange.Timestamp, service.Timestamp)
	assert.Equal(t, testHealthChange.ID, service.ID)
	assert.Equal(t, testHealthChange.Alive, service.Alive)
	assert.Equal(t, testHealthChange.TaskStatus, service.TaskStatus)
	assert.Equal(t, testHealthChange.AppID, service.AppID)
	assert.Equal(t, testHealthChange.Version, service.Version)
}