package events

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

// Event holds state of parsed fields from marathon EventStream
type SSEEvent struct {
	Type  string
	Body  []byte
	ID    string
	Delay string
}

var (
	lineFeed = []byte("\n")
	colon    = []byte{':'}
	space    = []byte{' '}
)

func (e *SSEEvent) parseLine(line []byte) bool {
	// https://www.w3.org/TR/2011/WD-eventsource-20110208/
	// Quote: Lines must be separated by either a U+000D CARRIAGE RETURN U+000A
	// LINE FEED (CRLF) character pair, a single U+000A LINE FEED (LF) character,
	// or a single U+000D CARRIAGE RETURN (CR) character.

	//If the line is empty (a blank line)
	if len(line) == 0 || bytes.Equal(line, lineFeed) {
		//Dispatch the event, as defined below.
		return !e.isEmpty()
	}

	//If the line starts with a U+003A COLON character (:)
	if bytes.HasPrefix(line, colon) {
		//Ignore the line.
		return false
	}

	var field string
	var value []byte
	//If the line contains a U+003A COLON character (:)
	//Collect the characters on the line before the first U+003A COLON character (:), and let field be that string.
	split := bytes.SplitN(line, colon, 2)
	if len(split) == 2 {
		field = string(split[0])
		//Collect the characters on the line after the first U+003A COLON character (:), and let value be that string.
		//If value starts with a U+0020 SPACE character, remove it from value.
		value = bytes.TrimPrefix(split[1], space)
	} else {
		//Otherwise, the string is not empty but does not contain a U+003A COLON character (:)
		//Process the field using the steps described below, using the whole line as the field name,
		//and the empty string as the field value.
		field = string(line)
		value = []byte{}

	}
	stringValue := string(value)
	//If the field name is
	switch field {
	case "event":
		//Set the event name buffer to field value.
		e.Type = stringValue
	case "data":
		//If the data buffer is not the empty string,
		if len(value) != 0 {
			//Append the field value to the data buffer,
			//then append a single U+000A LINE FEED (LF) character to the data buffer.
			e.Body = append(e.Body, value...)
			e.Body = append(e.Body, '\n')
		}
	case "id":
		//Set the last event ID buffer to the field value.
		e.ID = stringValue
	case "retry":
		e.Delay = stringValue
		// TODO consider reconnection delay
	}

	return false
}

func (e *SSEEvent) isEmpty() bool {
	return e.Type == "" && e.Body == nil && e.ID == ""
}

func (e *SSEEvent) String() string {
	return fmt.Sprintf("Type: %s, Body: %s", e.Type, string(e.Body))
}

func ParseSSEEvent(scanner *bufio.Scanner) (SSEEvent, error) {
	e := SSEEvent{}

	for dispatch := false; !dispatch; {
		if !scanner.Scan() {
			return e, io.EOF
		}
		line := scanner.Bytes()
		dispatch = e.parseLine(line)
		if err := scanner.Err(); err != nil {
			return e, scanner.Err()
		}
	}
	return e, nil
}

// ScanLines is higtly inspired by the function of the same name from bufio package,
// but is sensitive to CR as line separator
func ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	pos := lineTerminatorPosition(data)
	if pos != 0 {
		return pos + 1, dropCR(data[0:pos]), nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), dropCR(data), nil
	}
	// Request more data.
	return 0, nil, nil
}

func lineTerminatorPosition(data []byte) int {
	// https://www.w3.org/TR/2011/WD-eventsource-20110208/
	// Quote: Lines must be separated by either a U+000D CARRIAGE RETURN U+000A
	// LINE FEED (CRLF) character pair, a single U+000A LINE FEED (LF) character,
	// or a single U+000D CARRIAGE RETURN (CR) character.
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		// We have a full newline-terminated line
		return i
	} else if i := bytes.IndexByte(data, '\r'); i >= 0 {
		// We have a full CR terminated line
		return i
	}
	return 0
}

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}
