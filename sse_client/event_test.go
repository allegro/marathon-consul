package sse_client

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"strings"
)

func TestParseEvent_shouldParseLongDataField(t *testing.T) {
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

func TestParseEvent_shouldReturnErrOnEOFWhenEventIsNotReady(t *testing.T) {
	t.Parallel()
	// given
	events := bufio.NewReader(bytes.NewReader([]byte{}))

	// when
	_, err := parseEvent(events)

	// then
	assert.EqualError(t, err, "Unexpected EOF")
}

var testCases = []struct {
	stream   string
	expected []Event
}{
	{`: No Event`, []Event{}},
	{`: Multiline data
data: YHOO
data: +2
data: 10

`, []Event{{Data: b("YHOO\n+2\n10\n")}}},
	//TODO: Handle this case
	//{": Multiline CRLF/CR/LF\r\ndata:YHOO\rdata:+2\ndata:10\n", []Event{{Data: b("YHOO\n+2\n10\n")}}},
	{`: Test stream

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
	{`: Stream fires just one event
data

data
data

data: xyz
`,
		[]Event{{Data: b("xyz\n")}}},
	{`: Stream fires just one event
data: xyz

(0x9f2f00,0xc420014700)

`,
		[]Event{{Data: b("xyz\n")}}},
	{`: Stream fires two identical events
data:test

data: test
`,
		[]Event{
			{Data: b("test\n")},
			{Data: b("test\n")},
		}},
	{`: Full Event
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

func TestParseEvent(t *testing.T) {
	for _, tc := range testCases {
		t.Run(strings.Split(tc.stream, "\n")[0], func(t *testing.T) {
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
		})
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
