package sse_client

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/url"
)

type EventSource interface {
	//Open starts reading form source
	Open()
	//Close connection
	Close()
}

type client struct {
	request   *http.Request
	client    *http.Client
	close     context.CancelFunc
	onMessage func(Event)
	onError   func(error)
}

func NewEventSource(url *url.URL, httpClient *http.Client, onMessage func(Event), onError func(error)) EventSource {
	req := &http.Request{
		Method:     http.MethodGet,
		URL:        url,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       nil,
		Host:       url.Host,
	}
	req.Header.Add("Accept", "text/event-stream")
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)
	return &client{
		request:   req,
		client:    httpClient,
		onMessage: onMessage,
		onError:   onError,
		close:     cancel,
	}
}

func (c client) Open() {
	response, err := c.client.Do(c.request)
	if err != nil {
		c.onError(err)
		return
	}

	go func() {
		defer response.Body.Close()
		//TODO: Consider using bufio.Scanner
		reader := bufio.NewReader(response.Body)
		for {
			e, err := parseEvent(reader)
			if err != nil {
				if err == io.EOF {
					c.onMessage(e)
				}
				c.onError(err)
				return
			}
			c.onMessage(e)
		}
	}()
}

func (c client) Close() {
	c.close()
}
