package sse_client

import (
	"testing"

	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
)

func TestEventSource_IntegrationTest(t *testing.T) {
	t.Parallel()
	// given
	closeChan := make(chan struct{})
	dataChan := make(chan string, 2)
	server, transport := stubServer("/v2/events", dataChan, closeChan)
	defer server.Close()
	client := http.DefaultClient
	client.Transport = transport
	eventChan := make(chan Event, 2)
	errorChan := make(chan error, 2)
	onMessage := func(e Event) {
		eventChan <- e
	}
	onError := func(e error) {
		errorChan <- e
	}

	// when
	u, _ := url.Parse(server.URL + "/v2/events")
	es, err := NewEventSource(u, client, onMessage, onError)

	// then
	assert.NoError(t, err)

	// when
	dataChan <- "uno"
	dataChan <- "duo"
	err = es.Open()

	e := <-eventChan
	assert.Equal(t, "uno\n", string(e.Data))
	e = <-eventChan
	assert.Equal(t, "duo\n", string(e.Data))

	// when
	closeChan <- struct{}{}

	// then
	err = <-errorChan
	assert.EqualError(t, err, "Unexpected EOF")

	// when
	dataChan <- "tre"
	dataChan <- "quatro"
	err = es.Open()

	// then
	assert.NoError(t, err)
	e = <-eventChan
	assert.Equal(t, "tre\n", string(e.Data))
	e = <-eventChan
	assert.Equal(t, "quatro\n", string(e.Data))

	// when
	es.Close()

	// then
	err = <-errorChan
	assert.EqualError(t, err, "net/http: request canceled")

	// cleanup
	closeChan <- struct{}{}
}

// http://keighl.com/post/mocking-http-responses-in-golang/
func stubServer(uri string, data <-chan string, close <-chan struct{}) (*httptest.Server, *http.Transport) {
	return mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() == uri {
			f, _ := w.(http.Flusher)
			for {
				select {
				case d := <-data:
					fmt.Fprintf(w, "data: %s\n\n", d)
					f.Flush()
				case <-close:
					return
				}
			}
		} else {
			w.WriteHeader(404)
		}
	})
}

// TODO: Move to testutil
func mockServer(handle func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, *http.Transport) {
	server := httptest.NewServer(http.HandlerFunc(handle))

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	return server, transport
}
