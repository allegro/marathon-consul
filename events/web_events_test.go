package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebEventTypeValid(t *testing.T) {
	t.Parallel()
	out, err := ParseEvent([]byte(`{"eventType":"test","timestamp": "2014-03-01T23:29:30.158Z"}`))
	assert.Nil(t, err)
	assert.Equal(t, "2014-03-01T23:29:30.158Z", out.Timestamp.String())
}

func TestWebEventTypeInvalid(t *testing.T) {
	t.Parallel()

	out, err := ParseEvent([]byte(`{}`))
	assert.Error(t, err)
	assert.Equal(t, out, WebEvent{})
}

func TestTimestampInvalid(t *testing.T) {
	t.Parallel()
	out, err := ParseEvent([]byte(`{"eventType":"test","timestamp": "invalid"}`))
	assert.Error(t, err)
	assert.Equal(t, out, WebEvent{})
}

func TestMissingTimestamp(t *testing.T) {
	t.Parallel()
	out, err := ParseEvent([]byte(`{"eventType":"test"}`))
	assert.EqualError(t, err, "Missing timestamp")
	assert.Equal(t, out, WebEvent{})
}

func TestWebEventTypeInvalidJson(t *testing.T) {
	t.Parallel()

	out, err := ParseEvent([]byte(`not a json`))
	assert.Error(t, err)
	assert.Equal(t, out, WebEvent{})
}
