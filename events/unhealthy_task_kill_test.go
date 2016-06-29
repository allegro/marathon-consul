package events

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

var unhealthyTaskKill = &UnhealthyTaskKilled{
	AppID: "/my-app",
	ID:    "my-app_0-1396592784349",
}

func TestUnhealthTaskKillParseTask(t *testing.T) {
	t.Parallel()

	jsonified, err := json.Marshal(unhealthyTaskKill)
	assert.Nil(t, err)

	service, err := ParseUnhealthyTaskKilled(jsonified)
	assert.Nil(t, err)

	assert.Equal(t, unhealthyTaskKill.Timestamp, service.Timestamp)
	assert.Equal(t, unhealthyTaskKill.ID, service.ID)
	assert.Equal(t, unhealthyTaskKill.AppID, service.AppID)
}
