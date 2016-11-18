package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventTypeValid(t *testing.T) {
	t.Parallel()
	out, err := ParseEvent([]byte(`{"eventType":"test","timestamp": "2014-03-01T23:29:30.158Z"}`))
	assert.Nil(t, err)
	assert.Equal(t, "2014-03-01T23:29:30.158Z", out.Timestamp.String())
}

func TestEventTypeInvalid(t *testing.T) {
	t.Parallel()

	out, err := ParseEvent([]byte(`{}`))
	assert.Error(t, err)
	assert.Equal(t, out, Event{})
}

func TestTimestampInvalid(t *testing.T) {
	t.Parallel()
	out, err := ParseEvent([]byte(`{"eventType":"test","timestamp": "invalid"}`))
	assert.Error(t, err)
	assert.Equal(t, out, Event{})
}

func TestMissingTimestamp(t *testing.T) {
	t.Parallel()
	out, err := ParseEvent([]byte(`{"eventType":"test"}`))
	assert.EqualError(t, err, "Missing timestamp")
	assert.Equal(t, out, Event{})
}

func TestEventTypeInvalidJson(t *testing.T) {
	t.Parallel()

	out, err := ParseEvent([]byte(`not a json`))
	assert.Error(t, err)
	assert.Equal(t, out, Event{})
}
