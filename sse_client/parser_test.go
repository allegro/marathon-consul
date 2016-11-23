package sse_client

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors_shouldParseDataOnlyMessage(t *testing.T) {
	t.Parallel()
	// given
	events := bufio.NewReader(bytes.NewReader([]byte(`
: this is a test stream

data: some text


event: userconnect
data: {"username": "bobby", "time": "02:33:48"}

data: another message
data: with two lines
`)))

	// when
	event, err := parseEvent(events)

	// then
	assert.NoError(t, err)
	assert.Equal(t, "some text\n", string(event.Data))

	// when
	event, err = parseEvent(events)

	// then
	assert.NoError(t, err)
	assert.Equal(t, `{"username": "bobby", "time": "02:33:48"}`+"\n", string(event.Data))
	assert.Equal(t, "userconnect", string(event.Event))

	// when
	event, err = parseEvent(events)

	// then
	assert.EqualError(t, err, io.EOF.Error())
	assert.Equal(t, "another message\nwith two lines\n", string(event.Data))
}
