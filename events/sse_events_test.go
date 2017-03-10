package events

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvent_IfEventIsEmptyReturnsFalse(t *testing.T) {
	t.Parallel()
	// given
	event := &SSEEvent{
		Type: "status_update_event",
		Body: []byte(`{"id": "simpleId"}`),
		ID:   "id",
	}
	// when
	expected := false
	actual := event.isEmpty()
	// then
	assert.Equal(t, expected, actual)
}

func TestEvent_IfEventIsEmptyReturnsTrue(t *testing.T) {
	t.Parallel()
	// given
	event := &SSEEvent{}
	// when
	expected := true
	actual := event.isEmpty()
	// then
	assert.Equal(t, expected, actual)
}

func TestParseLine_WhenStautsUpdateEventPassed(t *testing.T) {
	t.Parallel()
	// given
	event := &SSEEvent{}
	line0 := []byte("id: 0")
	line1 := []byte("event: status_update_event")
	line2 := []byte("data: testData")
	expected0 := "0"
	expected1 := "status_update_event"
	expected2 := []byte("testData\n")
	// when
	event.parseLine(line0)
	event.parseLine(line1)
	event.parseLine(line2)
	// then
	assert.Equal(t, expected0, event.ID)
	assert.Equal(t, expected1, event.Type)
	assert.Equal(t, string(expected2), string(event.Body))
}

func TestParseLine_WhenGarbageIsProvidedBodyShouldBeNil(t *testing.T) {
	t.Parallel()
	// given
	event := &SSEEvent{}
	line := []byte("garbage data")
	expectedBody := []byte(nil)
	// when
	_ = event.parseLine(line)
	// then
	assert.Equal(t, expectedBody, event.Body)
}

func BenchmarkParseLine(b *testing.B) {
	// given
	longTestData := bytes.Repeat([]byte("testData"), 1)
	longLine := append([]byte("data: "), longTestData...)
	expectedEvent := &SSEEvent{Body: append(longTestData, []byte("\n")...)}

	var event *SSEEvent
	for i := 0; i <= b.N; i++ {
		event = &SSEEvent{}
		event.parseLine(longLine)
	}

	// then
	assert.Equal(b, string(expectedEvent.Body), string(event.Body))

}

var parseEventCases = []struct {
	in            string
	expectedEvent SSEEvent
}{
	{": No Event", SSEEvent{}},
	{"event: status_update_event\ndata: testData\n",
		SSEEvent{Type: "status_update_event", Body: []byte("testData\n")},
	},
	{"event: status_update_event\ndata: testData\ndummydata",
		SSEEvent{Type: "status_update_event", Body: []byte("testData\n")},
	},
	{"event: status_update_event\ndata",
		SSEEvent{Type: "status_update_event"},
	},
	{"event: status_update_event\ndata:\n",
		SSEEvent{Type: "status_update_event"},
	},
	{"event: some_event\ndata: abc\ndata: def",
		SSEEvent{Type: "some_event", Body: []byte("abc\ndef\n")},
	},
	{"event: some_event\ndata: aaa\ndata: ccc\ndata: 10",
		SSEEvent{Type: "some_event", Body: []byte("aaa\nccc\n10\n")},
	},
	{"event: some_event\ndata: abc\nid: 12",
		SSEEvent{Type: "some_event", Body: []byte("abc\n"), ID: "12"},
	},
	{"data: abc\n",
		SSEEvent{Body: []byte("abc\n")},
	},
}

func TestParseSSEEvent_TestCases(t *testing.T) {
	t.Parallel()

	for _, testCase := range parseEventCases {
		reader := strings.NewReader(testCase.in)
		sscanner := bufio.NewScanner(reader)
		// when
		actualEvent, _ := ParseSSEEvent(sscanner)
		// then
		assert.Equal(t, testCase.expectedEvent, actualEvent)

	}
}

var parseEventMultipleDataCases = []struct {
	in             string
	expectedEvents []SSEEvent
}{
	{"\n\n\n\n\n\n\n",
		[]SSEEvent{
			SSEEvent{},
		},
	},
	{"event: status_update_event\ndata: testData\n\nevent: some_event\ndata: someData",
		[]SSEEvent{
			SSEEvent{Type: "status_update_event", Body: []byte("testData\n")},
			SSEEvent{Type: "some_event", Body: []byte("someData\n")},
		},
	},
	{"event: status_update_event\ndata: testData\n\nid: 13\ndata: someData\n\nid: 14\ndata: abc\n\nid: 15\ndata: def\n",
		[]SSEEvent{
			SSEEvent{Type: "status_update_event", Body: []byte("testData\n")},
			SSEEvent{ID: "13", Body: []byte("someData\n")},
			SSEEvent{ID: "14", Body: []byte("abc\n")},
			SSEEvent{ID: "15", Body: []byte("def\n")},
		},
	},
	{"data: testData\n\ndata: someData\n\ndata: abc\n\ndata: def\n",
		[]SSEEvent{
			SSEEvent{Body: []byte("testData\n")},
			SSEEvent{Body: []byte("someData\n")},
			SSEEvent{Body: []byte("abc\n")},
			SSEEvent{Body: []byte("def\n")},
		},
	},
	{"data: testData\nretry: 10\ndummy: dummy field\n\ndata: someData\n\ndata: abc\n\ndata: def\n",
		[]SSEEvent{
			SSEEvent{Body: []byte("testData\n"), Delay: "10"},
			SSEEvent{Body: []byte("someData\n")},
			SSEEvent{Body: []byte("abc\n")},
			SSEEvent{Body: []byte("def\n")},
		},
	},
}

func TestParseSSEEvent_MultipleDataTestCases(t *testing.T) {
	t.Parallel()

	for _, testCase := range parseEventMultipleDataCases {

		reader := strings.NewReader(testCase.in)
		sscanner := bufio.NewScanner(reader)

		var err error
		var actualEvent SSEEvent
		for i := 0; len(testCase.expectedEvents) > i && err != fmt.Errorf("EOF"); i++ {
			// when
			actualEvent, err = ParseSSEEvent(sscanner)
			// then
			assert.Equal(t, testCase.expectedEvents[i], actualEvent)
		}

	}
}

func BenchmarkParseEvent(b *testing.B) {
	var event SSEEvent
	expectedEvent := SSEEvent{
		Type:  "some event type",
		ID:    "1",
		Body:  []byte("some data\nnext data\n"),
		Delay: "10",
	}

	for i := 0; i <= b.N; i++ {
		reader := strings.NewReader("event: some event type\nid: 1\ndata: some data\ndata: next data\nretry: 10\n")
		sscanner := bufio.NewScanner(reader)
		event, _ = ParseSSEEvent(sscanner)
	}

	assert.Equal(b, expectedEvent, event)

}

func TestParseSSEEvent_WhenVeryLongLineIsOnStream(t *testing.T) {
	t.Parallel()
	// given
	veryLongEventName := strings.Repeat("a", 1016)
	veryLongLine := fmt.Sprintf("event: %s\n", veryLongEventName)

	sreader := strings.NewReader(veryLongLine)
	sscanner := bufio.NewScanner(sreader)
	expectedEventType := veryLongEventName
	// when
	event, _ := ParseSSEEvent(sscanner)
	// then
	assert.Equal(t, expectedEventType, event.Type)
}

func TestParseSSEEvent_WhenVeryLongLineIsLongerThanMaxLineSize(t *testing.T) {
	t.Parallel()
	// given
	veryLongEventName := strings.Repeat("a", 10240)
	veryLongLine := fmt.Sprintf("event: %s\n\n", veryLongEventName)
	sreader := strings.NewReader(veryLongLine)
	sscanner := bufio.NewScanner(sreader)
	buffer := make([]byte, 1024)
	sscanner.Buffer(buffer, cap(buffer))
	expectedEventType := ""
	// when
	event, _ := ParseSSEEvent(sscanner)
	// then
	assert.Equal(t, expectedEventType, event.Type)
}

var scanLineCases = []struct {
	in              []byte
	atEOL           bool
	expectedAdvance int
	expectedToken   []byte
}{
	{[]byte("abcd\n"), false, 5, []byte("abcd")},
	{[]byte("abcd\r"), false, 5, []byte("abcd")},
	{[]byte("abcd\r\n"), false, 6, []byte("abcd")},
	{[]byte("abcd"), false, 0, []byte(nil)},
	{[]byte("abcd"), true, 4, []byte("abcd")},
	{[]byte("abcd\n"), true, 5, []byte("abcd")},
}

func TestScanLine_TestCases(t *testing.T) {
	t.Parallel()
	for _, testCase := range scanLineCases {
		// when
		advance, token, err := ScanLines(testCase.in, testCase.atEOL)
		// then
		require.NoError(t, err)
		assert.Equal(t, testCase.expectedAdvance, advance)
		assert.Equal(t, testCase.expectedToken, token)
	}
}
