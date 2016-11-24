package sse_client

import (
	"bufio"
	"net/http"
)

type EventSource interface {
	Read()
}

type client struct {
	request   http.Request
	client    http.Client
	onMessage func(Event)
	onError   func(error)
}

func NewEventSource(r http.Request, c http.Client, onMessage func(Event), onError func(error)) *client {
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
