package events

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEventTypeValid(t *testing.T) {
	t.Parallel()

	out, err := EventType([]byte(`{"eventType":"test"}`))
	assert.Nil(t, err)
	assert.Equal(t, out, "test")
}

func TestEventTypeInvalid(t *testing.T) {
	t.Parallel()

	out, err := EventType([]byte(`{}`))
	assert.Equal(t, err, ErrNoEvent)
	assert.Equal(t, out, "")
}

func TestParseEvent_APIPostEvent(t *testing.T) {
	t.Parallel()

	event := APIPostEvent{Type: "api_post_event"}
	jsonBlob, err := json.Marshal(event)
	assert.Nil(t, err)

	parsed, err := ParseEvent(jsonBlob)
	assert.Nil(t, err)
	assert.Equal(t, event, parsed.(APIPostEvent))
}

func TestParseEvent_DeploymentInfoEvent(t *testing.T) {
	event := DeploymentInfoEvent{Type: "deployment_info"}
	jsonBlob, err := json.Marshal(event)
	assert.Nil(t, err)

	parsed, err := ParseEvent(jsonBlob)
	assert.Nil(t, err)
	assert.Equal(t, event, parsed.(DeploymentInfoEvent))
}

func TestParseEvent_AppTerminatedEvent(t *testing.T) {
	event := AppTerminatedEvent{Type: "app_terminated_event"}
	jsonBlob, err := json.Marshal(event)
	assert.Nil(t, err)

	parsed, err := ParseEvent(jsonBlob)
	assert.Nil(t, err)
	assert.Equal(t, event, parsed.(AppTerminatedEvent))
}
