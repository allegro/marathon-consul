package marathon

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStreamer_RecoverShouldReturnErrorWhenCantStart(t *testing.T) {
	server, transport := stubServer("/v2/events", "")
	defer server.Close()
	client := &http.Client{Transport: transport}
	s := Streamer{client: client}
	s.Start()

	err := s.Recover()

	assert.EqualError(t, err, "Cannot recover Streamer: Subscription request errored: Get : unsupported protocol scheme \"\"")
}

func TestStreamer_StartShouldReturnErrorOnInvalidUrl(t *testing.T) {
	s := Streamer{client: http.DefaultClient}

	err := s.Start()

	assert.EqualError(t, err, "Subscription request errored: Get : unsupported protocol scheme \"\"")
}

func TestStreamer_StartShouldReturnErrorOnNon200Response(t *testing.T) {
	server, transport := stubServer("/v2/events", "")
	defer server.Close()
	client := &http.Client{Transport: transport}
	s := Streamer{client: client, subURL: "http://marathon/invalid/path"}

	err := s.Start()

	assert.EqualError(t, err, "Event stream not connected: Expected 200 but got 404")
}

func TestStreamer_StartClosesNon200ResponseBody(t *testing.T) {
	body := &trackedBody{}
	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Status:     "404 Not Found",
			Header:     make(http.Header),
			Body:       body,
			Request:    req,
		}, nil
	})}
	s := Streamer{client: client, subURL: "http://marathon/v2/events"}

	err := s.Start()

	assert.EqualError(t, err, "Event stream not connected: Expected 200 but got 404")
	assert.True(t, body.closed)
}

type trackedBody struct {
	closed bool
}

func (body *trackedBody) Read([]byte) (int, error) { return 0, io.EOF }
func (body *trackedBody) Close() error {
	body.closed = true
	return nil
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (roundTripper roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return roundTripper(req)
}

func TestStreamer_StartShouldReturnNoErrorIfSuccessfulConnects(t *testing.T) {
	server, transport := stubServer("/v2/events", "")
	defer server.Close()
	client := &http.Client{Transport: transport}
	s := Streamer{client: client, subURL: "http://marathon/v2/events"}

	err := s.Start()

	assert.NoError(t, err)
}

func TestStreamer_RecoverShouldReturnNoErrorIfSuccessfulConnects(t *testing.T) {
	server, transport := stubServer("/v2/events", "")
	defer server.Close()
	client := &http.Client{Transport: transport}
	s := Streamer{client: client, subURL: "http://marathon/v2/events"}
	s.Start()

	err := s.Recover()

	assert.NoError(t, err)
}

func TestStreamer_StopShouldPreventRecovery(t *testing.T) {
	server, transport := stubServer("/v2/events", "")
	defer server.Close()
	client := &http.Client{Transport: transport}
	s := Streamer{client: client, subURL: "http://marathon/v2/events"}

	s.Start()
	s.Stop()
	err := s.Recover()

	assert.Error(t, err, "Streamer is not recoverable")
}

func TestStreamer_StopShouldCancelRequest(t *testing.T) {
	ready := make(chan bool)
	wait := make(chan bool)
	server, transport := mockServer(func(w http.ResponseWriter, r *http.Request) {
		ready <- true
		<-wait
	})
	defer server.Close()
	client := &http.Client{Transport: transport}
	s := Streamer{client: client, subURL: "http://marathon/v2/events"}

	go func() {
		err := s.Start()
		assert.Error(t, err, "Subscription request errored: Get http://marathon/v2/events: net/http: request canceled")
	}()

	<-ready
	s.Stop()
	wait <- false
}
