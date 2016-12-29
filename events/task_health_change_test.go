package events

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testHealthChange = &TaskHealthChange{
	Timestamp: "2014-03-01T23:29:30.158Z",
	ID:        "my-app_0-1396592784349",
	Alive:     true,
	AppID:     "/my-app",
	Version:   "2014-04-04T06:26:23.051Z",
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
	assert.Equal(t, testHealthChange.AppID, service.AppID)
	assert.Equal(t, testHealthChange.Version, service.Version)
}

func TestHealthChangeParseTaskWithoutData(t *testing.T) {
	t.Parallel()

	event, err := ParseTaskHealthChange([]byte("{}"))
	assert.Nil(t, event)
	assert.EqualError(t, err, "Missing task ID")
}

func TestHealthChangeParseTaskWithBrokenJson(t *testing.T) {
	t.Parallel()

	event, err := ParseTaskHealthChange([]byte("not a Json"))
	assert.Nil(t, event)
	assert.Error(t, err)
}

func TestInstanceIDToTaskID(t *testing.T) {
	t.Parallel()

	ids := []string{
		"python1.marathon-0bf4660a-cdc0-11e6-87df-0242ee53bf4b",
		"python1.instance-0bf4660a-cdc0-11e6-87df-0242ee53bf4b",
		"python1.0bf4660a-cdc0-11e6-87df-0242ee53bf4b",
	}

	for _, id := range ids {
		instance := TaskHealthChange{InstanceID: id}
		assert.Equal(t, "python1.0bf4660a-cdc0-11e6-87df-0242ee53bf4b", instance.TaskID().String())
	}
}
