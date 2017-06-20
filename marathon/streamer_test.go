package marathon

import (
	"testing"

	"net/http"

	"io/ioutil"

	"github.com/stretchr/testify/assert"
)

func TestStreamer_RecoverShouldReturnErrorWhenCantStart(t *testing.T) {
	server, transport := stubServer("/v2/events", "")
	defer server.Close()
	client := &http.Client{Transport: transport}
	s := streamer{client: client}
	s.Start()

	r, err := s.Recover()

	assert.Nil(t, r)
	assert.EqualError(t, err, "Cannot recover Streamer: Subscription request errored: Get : unsupported protocol scheme \"\"")
}

func TestStreamer_StartShouldReturnErrorOnInvalidUrl(t *testing.T) {
	s := streamer{client: http.DefaultClient}

	r, err := s.Start()

	assert.Nil(t, r)
	assert.EqualError(t, err, "Subscription request errored: Get : unsupported protocol scheme \"\"")
}

func TestStreamer_StartShouldReturnErrorOnNon200Response(t *testing.T) {
	server, transport := stubServer("/v2/events", "")
	defer server.Close()
	client := &http.Client{Transport: transport}
	s := streamer{client: client, subURL: "http://marathon/invalid/path"}

	r, err := s.Start()

	assert.Nil(t, r)
	assert.EqualError(t, err, "Event stream not connected: Expected 200 but got 404")
}

func TestStreamer_StartShouldReturnNoErrorIfSuccessfulConnects(t *testing.T) {
	server, transport := stubServer("/v2/events", "OK")
	defer server.Close()
	client := &http.Client{Transport: transport}
	s := streamer{client: client, subURL: "http://marathon/v2/events"}

	r, err := s.Start()
	bytes, err := ioutil.ReadAll(r)

	assert.Equal(t, "OK\n", string(bytes))
	assert.NoError(t, err)
}

func TestStreamer_RecoverShouldReturnNoErrorIfSuccessfulConnects(t *testing.T) {
	server, transport := stubServer("/v2/events", "")
	defer server.Close()
	client := &http.Client{Transport: transport}
	s := streamer{client: client, subURL: "http://marathon/v2/events"}
	s.Start()

	r, err := s.Recover()

	assert.NotNil(t, r)
	assert.NoError(t, err)
}

func TestStreamer_StopShouldPreventRecovery(t *testing.T) {
	server, transport := stubServer("/v2/events", "")
	defer server.Close()
	client := &http.Client{Transport: transport}
	s := streamer{client: client, subURL: "http://marathon/v2/events"}

	s.Start()
	s.Stop()
	r, err := s.Recover()

	assert.Nil(t, r)
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
	s := streamer{client: client, subURL: "http://marathon/v2/events"}

	go func() {
		r, err := s.Start()

		assert.Nil(t, r)
		assert.Error(t, err, "Subscription request errored: Get http://marathon/v2/events: net/http: request canceled")
	}()

	<-ready
	s.Stop()
	wait <- false
}
