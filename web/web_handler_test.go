package web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const maxEventSize = 4096

func TestWebHandler_Send202AcceptedWhenEventsPassesValidation(t *testing.T) {
	t.Parallel()

	// given
	body := []byte(`{
		  "eventType":"status_update_event",
		  "timestamp":"2015-12-07T09:33:40.898Z"
		}`)

	queue := make(chan event, 1)
	handler := newWebHandler(queue, maxEventSize)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer(body))
	recorder := httptest.NewRecorder()

	// when
	handler.Handle(recorder, req)

	// then
	assertAccepted(t, recorder)
}

func TestWebHandler_HandleReaderError(t *testing.T) {
	t.Parallel()

	// given
	handler := newWebHandler(nil, maxEventSize)
	req, _ := http.NewRequest("POST", "/events", BadReader{})

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "Some error\n", recorder.Body.String())
}

func TestWebHandler_DropBigBody(t *testing.T) {
	t.Parallel()

	// given
	handler := newWebHandler(nil, maxEventSize)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer(make([]byte, 4097)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "http: request body too large\n", recorder.Body.String())
}

func TestWebHandler_DropEmptyBody(t *testing.T) {
	t.Parallel()

	// given
	handler := newWebHandler(nil, maxEventSize)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte{}))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assertDropped(t, recorder)
}

func TestWebHandler_DropAppInvalidBody(t *testing.T) {
	t.Parallel()

	// given
	handler := newWebHandler(nil, maxEventSize)
	body := `{"type":  "app_terminated_event", "appID": 123}`
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assertDropped(t, recorder)
}

func TestWebHandler_DropMalformedEventType(t *testing.T) {
	t.Parallel()

	// given
	handler := newWebHandler(nil, maxEventSize)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(`{eventType:"test_event"}`)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assertDropped(t, recorder)
}

func TestWebHandler_DropInvalidEventType(t *testing.T) {
	t.Parallel()

	// given
	handler := newWebHandler(nil, maxEventSize)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(`{"eventType":[1,2]}`)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assertDropped(t, recorder)
}

func TestWebHandler_DropUnknownEventType(t *testing.T) {
	t.Parallel()

	// given
	handler := newWebHandler(nil, maxEventSize)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(`{"eventType":"test_event"}`)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assertDropped(t, recorder)
}

func assertAccepted(t *testing.T, recorder *httptest.ResponseRecorder) {
	assert.Equal(t, 202, recorder.Code)
	assert.Equal(t, "OK\n", recorder.Body.String())
}

func assertDropped(t *testing.T, recorder *httptest.ResponseRecorder) {
	assert.Equal(t, 200, recorder.Code)
}
