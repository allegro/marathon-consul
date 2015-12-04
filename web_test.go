package main

import (
	"bytes"
	"encoding/json"
	"github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/events"
	"github.com/allegro/marathon-consul/marathon"
	. "github.com/allegro/marathon-consul/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequest("GET", "http://example.com/health", nil)
	assert.Nil(t, err)

	recorder := httptest.NewRecorder()
	HealthHandler(recorder, req)

	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "OK\n", recorder.Body.String())
}

func TestForwardHandler_NotHandleUnknownEventType(t *testing.T) {
	t.Parallel()
	// given
	handler := ForwardHandler{nil, nil}
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(`{"eventType":"test_event"}`)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "cannot handle test_event\n", recorder.Body.String())
}

func TestForwardHandler_NotHandleMalformedEventType(t *testing.T) {
	t.Parallel()
	// given
	handler := ForwardHandler{nil, nil}
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(`{eventType:"test_event"}`)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "invalid character 'e' looking for beginning of object key string\n", recorder.Body.String())
}

func TestForwardHandler_HandleMalformedEventType(t *testing.T) {
	t.Parallel()
	// given
	handler := ForwardHandler{nil, nil}
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(`{eventType:"test_event"}`)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "invalid character 'e' looking for beginning of object key string\n", recorder.Body.String())
}

func TestForwardHandler_NotHandleInvalidEventType(t *testing.T) {
	t.Parallel()
	// given
	handler := ForwardHandler{nil, nil}
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(`{"eventType":[1,2]}`)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "json: cannot unmarshal array into Go value of type string\n", recorder.Body.String())
}

func TestForwardHandler_HandleAppTerminatedEvent(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)
	service := consul.NewConsulStub()
	for _, task := range app.Tasks {
		service.Register(consul.MarathonTaskToConsulService(task, app.HealthChecks, app.Labels))
	}
	handler := ForwardHandler{service, marathon}
	body, _ := json.Marshal(events.AppTerminatedEvent{
		Type:  "app_terminated_event",
		AppID: app.ID,
	})
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	services, _ := service.GetAllServices()
	// then
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "OK\n", recorder.Body.String())
	assert.Empty(t, services)
}
