package events

import (
	"encoding/json"
	"github.com/allegro/marathon-consul/apps"
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

func TestEventTypeInvalidJson(t *testing.T) {
	t.Parallel()

	out, err := EventType([]byte(`not a json`))
	assert.Error(t, err)
	assert.Equal(t, out, "")
}

func TestParseEvent_InvalidJson(t *testing.T) {
	t.Parallel()

	parsed, err := ParseEvent([]byte(`not a json`))
	assert.Error(t, err)
	assert.Nil(t, parsed)
}

func TestParseEvent_APIPostEvent(t *testing.T) {
	t.Parallel()

	event := APIPostEvent{Type: "api_post_event", App: &apps.App{}}
	jsonBlob, err := json.Marshal(event)
	assert.Nil(t, err)

	parsed, err := ParseEvent(jsonBlob)
	assert.NoError(t, err)
	assert.Equal(t, "api_post_event", parsed.GetType())
	assert.Len(t, parsed.Apps(), 1)
	assert.Equal(t, event, parsed.(APIPostEvent))
}

func TestParseEvent_DeploymentInfoEvent(t *testing.T) {
	event := DeploymentInfoEvent{Type: "deployment_info"}
	jsonBlob, err := json.Marshal(event)
	assert.Nil(t, err)

	parsed, err := ParseEvent(jsonBlob)
	assert.NoError(t, err)
	assert.Equal(t, "deployment_info", parsed.GetType())
	assert.Empty(t, parsed.Apps())
	assert.Equal(t, event, parsed.(DeploymentInfoEvent))
}

func TestParseEvent_AppTerminatedEvent(t *testing.T) {
	event := AppTerminatedEvent{Type: "app_terminated_event", AppID: "appId"}
	jsonBlob, err := json.Marshal(event)
	assert.Nil(t, err)

	parsed, err := ParseEvent(jsonBlob)
	assert.NoError(t, err)
	assert.Equal(t, "app_terminated_event", parsed.GetType())
	assert.Equal(t, []*apps.App{&apps.App{ID: "appId"}}, parsed.Apps())
	assert.Equal(t, event, parsed.(AppTerminatedEvent))
}
