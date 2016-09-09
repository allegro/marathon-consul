package events

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	assert.Error(t, err)
	assert.Equal(t, out, "")
}

func TestEventTypeInvalidJson(t *testing.T) {
	t.Parallel()

	out, err := EventType([]byte(`not a json`))
	assert.Error(t, err)
	assert.Equal(t, out, "")
}
