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
	f := fixture()

	t.Run("Client opens connection and reads events", func(t *testing.T) {
		// when
		f.send("uno")
		f.send("duo")
		f.eventSource.Open()
		// then
		assert.Equal(t, "uno\n", string(f.event().Data))
		assert.Equal(t, "duo\n", string(f.event().Data))
	})

	t.Run("Server closes connection", func(t *testing.T) {
		// when
		f.closeResponse()
		// then
		assert.EqualError(t, f.err(), "Unexpected EOF")
	})

	t.Run("Client reopens closed connection", func(t *testing.T) {
		// when
		f.send("tre")
		f.send("quatro")
		f.eventSource.Open()
		// then
		assert.Equal(t, "tre\n", string(f.event().Data))
		assert.Equal(t, "quatro\n", string(f.event().Data))
	})

	t.Run("Client closes connection", func(t *testing.T) {
		// when
		f.eventSource.Close()
		// then
		assert.EqualError(t, f.err(), "net/http: request canceled")
	})

	// cleanup
	f.closeResponse()
}

type testFixture struct {
	eventSource EventSource
	server      *httptest.Server
	closeChan   chan struct{}
	dataChan    chan string
	eventChan   chan Event
	errorChan   chan error
}

func (f *testFixture) send(data string) {
	f.dataChan <- data
}

func (f *testFixture) event() Event {
	return <-f.eventChan
}

func (f *testFixture) err() error {
	return <-f.errorChan
}

func (f *testFixture) closeResponse() {
	f.closeChan <- struct{}{}
}

func fixture() (f testFixture) {
	f.closeChan = make(chan struct{})
	f.dataChan = make(chan string, 2)
	client := http.DefaultClient
	f.server, client.Transport = stubServer("/v2/events", f.dataChan, f.closeChan)
	f.eventChan = make(chan Event, 2)
	f.errorChan = make(chan error, 2)
	onMessage := func(e Event) {
		f.eventChan <- e
	}
	onError := func(e error) {
		f.errorChan <- e
	}

	u, _ := url.Parse(f.server.URL + "/v2/events")
	f.eventSource = NewEventSource(u, client, onMessage, onError)
	return f
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
