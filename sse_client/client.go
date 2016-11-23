package sse_client

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"
)

type SSEClient interface {
	Read()
}

type client struct {
	request   http.Request
	client    http.Client
	onMessage func(Event)
	onError   func(error)
}

func NewSSEClient(r http.Request, c http.Client, onMessage func(Event), onError func(error)) *client {
	return &client{request: r, client: c, onMessage: onMessage, onError: onError}
}

func (c client) Read() {
	go func() {
		req, err := http.NewRequest(http.MethodGet, "marathon-dev.qxlint/v2/events", nil)
		if err != nil {
			c.onError(err)
		}
		req.Header.Set("Accept", "text/event-stream")

		res, err := c.client.Do(req)
		if err != nil {
			c.onError(err)
		}
		//TODO: Consider using bufio.Scanner
		reader := bufio.NewReader(res.Body)
		defer res.Body.Close()
		for {
			e, err := parseEvent(reader)
			if err != nil {
				c.onError(err)
				return
			}
			c.onMessage(e)
		}
	}()
}

type Event struct {
	ID             string
	Event          string
	Data           []byte
	ReconnectDelay uint64
}

func parseEvent(reader *bufio.Reader) (Event, error) {
	e := Event{}
	for dispatch := false; dispatch == false; {
		line, err := reader.ReadBytes('\n')
		if err == io.EOF {
			dispatch = e.parseLine(line)
			if !dispatch {
				return e, errors.New("Unexpected EOF")
			}
			return e, io.EOF
		}
		if err != nil {
			return e, err
		}
		dispatch = e.parseLine(line)
	}
	return e, nil
}

// parseLine is implementation of event stream interpretation w3c standard. Is designed to be run multiple time.
// `line` argument should be line readed from buffer without new line character.
// Returns true when event is ready for dispatch
// See: https://www.w3.org/TR/eventsource/#event-stream-interpretation
// Reference implementation: https://github.com/WebKit/webkit/blob/9f191f/Source/WebCore/page/EventSource.cpp#L272-L363
func (e *Event) parseLine(line []byte) bool {

	line = bytes.TrimSuffix(line, []byte{'\n'})

	//If the line is empty (a blank line)
	if len(line) == 0 {
		//Dispatch the event, as defined below.
		return !e.isEmpty()
	}

	//If the line starts with a U+003A COLON character (:)
	if bytes.HasPrefix(line, []byte{':'}) {
		//Ignore the line.
		return false
	}

	var field string
	var value []byte
	//If the line contains a U+003A COLON character (:)
	//Collect the characters on the line before the first U+003A COLON character (:), and let field be that string.
	split := bytes.SplitN(line, []byte{':'}, 2)
	if len(split) == 2 {
		field = string(split[0])
		//Collect the characters on the line after the first U+003A COLON character (:), and let value be that string.
		//If value starts with a U+0020 SPACE character, remove it from value.
		value = bytes.TrimPrefix(split[1], []byte{' '})
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
		e.Event = stringValue
	case "data":
		//Append the field value to the data buffer, then append a single U+000A LINE FEED (LF) character to the data buffer.
		e.Data = append(e.Data, value...)
		e.Data = append(e.Data, '\n')
	case "id":
		//Set the last event ID buffer to the field value.
		e.ID = stringValue
	case "retry":
		//If the field value consists of only characters in the range U+0030 DIGIT ZERO (0) to U+0039 DIGIT NINE (9),
		reconnectDelay, err := strconv.ParseUint(stringValue, 10, 64)
		if err == nil {
			//then interpret the field value as an integer in base ten, and set the event stream's reconnection time to that integer.
			e.ReconnectDelay = reconnectDelay
		} else {
			//Otherwise, ignore the field.
		}
	}

	return false
}

func (e *Event) isEmpty() bool {
	return e.ID == "" && e.ReconnectDelay == 0 && e.Event == "" && e.Data == nil
}
