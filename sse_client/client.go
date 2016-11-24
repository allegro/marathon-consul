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
	//Check if connection is active
	IsActive() bool
}

type client struct {
	request   *http.Request
	client    *http.Client
	close     func()
	onMessage func(Event)
	onError   func(error)
}

func NewEventSource(url *url.URL, httpClient *http.Client, onMessage func(Event), onError func(error)) (*client, error) {
	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	return &client{
		request:   req,
		client:    httpClient,
		onMessage: onMessage,
		onError:   onError,
		close:     cancel,
	}, err
}

func (c client) Open() error {
	response, err := c.client.Do(c.request)
	if err != nil {
		c.onError(err)
		return err
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

	return nil
}

func (c client) Close() {
	c.close()
}
