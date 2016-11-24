package sse_client

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrors_shouldParseLongDataField(t *testing.T) {
	t.Parallel()
	// given
	data := make([]byte, 10*1024*1024)
	for i := range data {
		data[i] = '-'
	}
	data = append(data, '\n')
	message := append(b(`data:`), data...)
	events := bufio.NewReader(bytes.NewReader(message))

	// when
	event, err := parseEvent(events)

	// then
	assert.EqualError(t, err, io.EOF.Error())
	assert.Equal(t, string(data), string(event.Data))
}

func TestErrors_shouldReturnErrOnEOFWhenEventIsNotReady(t *testing.T) {
	t.Parallel()
	// given
	events := bufio.NewReader(bytes.NewReader([]byte{}))

	// when
	_, err := parseEvent(events)

	// then
	assert.EqualError(t, err, "Unexpected EOF")
}

func TestParseEvent(t *testing.T) {
	testCases := []struct {
		stream   string
		expected []Event
	}{
		{`data: YHOO
data: +2
data: 10

`, []Event{{Data: b("YHOO\n+2\n10\n")}}},
		{`: test stream

data: first event
id: 1

data: second event
id

data: third event
`,
			[]Event{
				{ID: "1", Data: b("first event\n")},
				{Data: b("second event\n")},
				{Data: b("third event\n")},
			}},
		{`:The following stream fires just one event:
data

data
data

data: xyz
`,
			[]Event{{Data: b("xyz\n")}}},
		{`: The following stream fires two identical events:
data:test

data: test
`,
			[]Event{
				{Data: b("test\n")},
				{Data: b("test\n")},
			}},
		{`: The Full Event
event: Ionizing
event: radiation
id: U+2622
data:╔═╗
data:║☢║
data:╚═╝
warn: RADIOACTIVE
retry: 10
Radioactive waste...
`,
			[]Event{
				{ID: "U+2622", Event: "radiation", Data: b("╔═╗\n║☢║\n╚═╝\n"), ReconnectDelay: 10},
			}},
	}
	for _, tc := range testCases {
		events := bufio.NewReader(bytes.NewReader(b(tc.stream)))
		for _, e := range tc.expected {
			event, err := parseEvent(events)
			if err != io.EOF {
				assert.NoError(t, err)
			}
			assertEqual(t, e, event)
		}
		event, err := parseEvent(events)
		assert.EqualError(t, err, "Unexpected EOF")
		assertEqual(t, Event{}, event)
	}
}

func assertEqual(t *testing.T, expected, actual Event) {
	assert.Equal(t, expected.Event, actual.Event)
	assert.Equal(t, expected.ID, actual.ID)
	assert.Equal(t, expected.ReconnectDelay, actual.ReconnectDelay)
	assert.Equal(t, string(expected.Data), string(actual.Data))
}

func b(s string) []byte {
	return []byte(s)
}
