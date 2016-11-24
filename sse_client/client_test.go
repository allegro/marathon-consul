package sse_client

import (
	"testing"

	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
)

func TestEventSource_SimpleEvents(t *testing.T) {
	t.Parallel()
	// given
	server, transport := stubServer("/v2/events", "data: abc\n\ndata: xyz\n")
	defer server.Close()
	client := http.DefaultClient
	client.Transport = transport
	eventChan := make(chan Event)
	errorChan := make(chan error)
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
	err = es.Open()

	// then
	e := <-eventChan
	assert.Equal(t, "abc\n", string(e.Data))
	e = <-eventChan
	assert.Equal(t, "xyz\n", string(e.Data))

	// then
	err = <-errorChan
	assert.Error(t, err)
}

// http://keighl.com/post/mocking-http-responses-in-golang/
func stubServer(uri string, body string) (*httptest.Server, *http.Transport) {
	return mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() == uri {
			fmt.Fprint(w, body)
		} else {
			w.WriteHeader(404)
		}
	})
}

func mockServer(handle func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, *http.Transport) {
	server := httptest.NewServer(http.HandlerFunc(handle))

	transport := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(server.URL)
		},
	}

	return server, transport
}
