package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/events"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/tasks"
	. "github.com/allegro/marathon-consul/utils"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestForwardHandler_NotHandleUnknownEventType(t *testing.T) {
	t.Parallel()
	// given
	handler := NewEventHandler(nil, nil)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(`{"eventType":"test_event"}`)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "Cannot handle test_event\n", recorder.Body.String())
}

func TestForwardHandler_HandleRadderError(t *testing.T) {
	t.Parallel()
	// given
	handler := NewEventHandler(nil, nil)
	req, _ := http.NewRequest("POST", "/events", BadReader{})
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "Some error\n", recorder.Body.String())
}

func TestForwardHandler_HandleEmptyBody(t *testing.T) {
	t.Parallel()
	// given
	handler := NewEventHandler(nil, nil)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte{}))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "unexpected end of JSON input\n", recorder.Body.String())
}

func TestForwardHandler_NotHandleMalformedEventType(t *testing.T) {
	t.Parallel()
	// given
	handler := NewEventHandler(nil, nil)
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
	handler := NewEventHandler(nil, nil)
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
	handler := NewEventHandler(nil, nil)
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
	handler := NewEventHandler(service, marathon)
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

func TestForwardHandler_HandleAppInvalidBody(t *testing.T) {
	t.Parallel()
	// given
	handler := NewEventHandler(nil, nil)
	body := `{"type":  "app_terminated_event", "appID": 123}`
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "no event\n", recorder.Body.String())
}

func TestForwardHandler_HandleAppTerminatedEventInvalidBody(t *testing.T) {
	t.Parallel()
	// given
	handler := NewEventHandler(nil, nil)
	body := `{"appId":"/python/simple","eventType":"app_terminated_event","timestamp":2015}`
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "json: cannot unmarshal number into Go value of type string\n", recorder.Body.String())
}

func TestForwardHandler_HandleAppTerminatedEventForUnknownApp(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)
	handler := NewEventHandler(nil, marathon)
	body := `{"appId":"/unknown/app","eventType":"app_terminated_event","timestamp":"2015-12-07T09:02:49.934Z"}`
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	// then
	assert.Equal(t, 500, recorder.Code)
	assert.Equal(t, "app not found\n", recorder.Body.String())
}

func TestForwardHandler_HandleAppTerminatedEventWithProblemsOnDeregistering(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)
	service := consul.NewConsulStub()
	for _, task := range app.Tasks {
		service.Register(consul.MarathonTaskToConsulService(task, app.HealthChecks, app.Labels))
	}
	service.ErrorServices["/test/app.1"] = fmt.Errorf("Cannot deregister service")
	service.ErrorServices["/test/app.2"] = fmt.Errorf("Cannot deregister service")
	handler := NewEventHandler(service, marathon)
	body, _ := json.Marshal(events.AppTerminatedEvent{
		Type:  "app_terminated_event",
		AppID: app.ID,
	})
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	// then
	assert.Equal(t, 500, recorder.Code)
	assert.Equal(t, "2 errors occured deregistering 3 services:\n0: Cannot deregister service\n1: Cannot deregister service\n", recorder.Body.String())
	assert.Len(t, service.RegisteredServicesIds(), 2)
	assert.NotContains(t, "test/app.0", service.RegisteredServicesIds())
	assert.Contains(t, service.RegisteredServicesIds(), "/test/app.1")
	assert.Contains(t, service.RegisteredServicesIds(), "/test/app.2")
}

func TestForwardHandler_NotHandleStatusEventWithInvalidBody(t *testing.T) {
	t.Parallel()
	// given
	handler := NewEventHandler(nil, nil)
	body := `{
	  "slaveId":"85e59460-a99e-4f16-b91f-145e0ea595bd-S0",
	  "taskId":"python_simple.4a7e99d0-9cc1-11e5-b4d8-0a0027000004",
	  "taskStatus":"TASK_KILLED",
	  "message":"",
	  "appId":"/test/app",
	  "host":"localhost",
	  "ports": 31372,
	  "version":"2015-12-07T09:02:48.981Z",
	  "eventType":"status_update_event",
	  "timestamp":"2015-12-07T09:02:49.934Z"
	}`
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "json: cannot unmarshal number into Go value of type []int\n",
		recorder.Body.String())
}

func TestForwardHandler_NotHandleStatusEventAboutStartingTask(t *testing.T) {
	t.Parallel()
	// given
	handler := NewEventHandler(nil, nil)
	ignoringTaskStatuses := []string{"TASK_STAGING", "TASK_STARTING", "TASK_RUNNING", "unknown"}
	for _, taskStatus := range ignoringTaskStatuses {
		body := `{
		  "slaveId":"85e59460-a99e-4f16-b91f-145e0ea595bd-S0",
		  "taskId":"python_simple.4a7e99d0-9cc1-11e5-b4d8-0a0027000004",
		  "taskStatus":"` + taskStatus + `",
		  "message":"",
		  "appId":"/test/app",
		  "host":"localhost",
		  "ports":[
			31372
		  ],
		  "version":"2015-12-07T09:02:48.981Z",
		  "eventType":"status_update_event",
		  "timestamp":"2015-12-07T09:02:49.934Z"
		}`
		req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))
		// when
		recorder := httptest.NewRecorder()
		handler.Handle(recorder, req)
		// then
		assert.Equal(t, 400, recorder.Code)
		assert.Equal(t, "Not Handling task python_simple.4a7e99d0-9cc1-11e5-b4d8-0a0027000004 with status "+taskStatus+"\n",
			recorder.Body.String())
	}
}

func TestForwardHandler_HandleStatusEventAboutDeadTask(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)
	service := consul.NewConsulStub()
	for _, task := range app.Tasks {
		service.Register(consul.MarathonTaskToConsulService(task, app.HealthChecks, app.Labels))
	}
	handler := NewEventHandler(service, marathon)
	taskStatuses := []string{"TASK_FINISHED", "TASK_FAILED", "TASK_KILLED", "TASK_LOST"}
	for _, taskStatus := range taskStatuses {
		body := `{
		  "slaveId":"85e59460-a99e-4f16-b91f-145e0ea595bd-S0",
		  "taskId":"` + app.Tasks[1].ID.String() + `",
		  "taskStatus":"` + taskStatus + `",
		  "message":"Command terminated with signal Terminated",
		  "appId":"/test/app",
		  "host":"localhost",
		  "ports":[
			31372
		  ],
		  "version":"2015-12-07T09:02:48.981Z",
		  "eventType":"status_update_event",
		  "timestamp":"2015-12-07T09:33:40.898Z"
		}`
		req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))
		// when
		recorder := httptest.NewRecorder()
		handler.Handle(recorder, req)
		servicesIds := service.RegisteredServicesIds()
		// then
		assert.Equal(t, 200, recorder.Code)
		assert.Equal(t, "OK\n", recorder.Body.String())
		assert.Len(t, servicesIds, 2)
		assert.NotContains(t, servicesIds, app.Tasks[1].ID)
		assert.Contains(t, servicesIds, app.Tasks[0].ID.String())
		assert.Contains(t, servicesIds, app.Tasks[2].ID.String())
	}
}

func TestForwardHandler_NotHandleHealthStatusEventWhenAppHasNotConsulLabel(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("/test/app", 3)
	app.Labels["consul"] = "false"
	marathon := marathon.MarathonerStubForApps(app)
	service := consul.NewConsulStub()
	handler := NewEventHandler(service, marathon)
	body := healthStatusChangeEventForTask("/test/app.1")
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	servicesIds := service.RegisteredServicesIds()
	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "/test/app is not consul app. Missing consul:true label\n", recorder.Body.String())
	assert.Len(t, servicesIds, 0)
}

func TestForwardHandler_HandleHealthStatusEvent(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)
	service := consul.NewConsulStub()
	handler := NewEventHandler(service, marathon)
	body := healthStatusChangeEventForTask("/test/app.1")
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	servicesIds := service.RegisteredServicesIds()
	// then
	assert.Equal(t, 200, recorder.Code)
	assert.Equal(t, "OK\n", recorder.Body.String())
	assert.Len(t, servicesIds, 1)
	assert.Contains(t, servicesIds, app.Tasks[1].ID.String())
}

func TestForwardHandler_HandleHealthStatusEventWithErrorsOnRegistration(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)
	service := consul.NewConsulStub()
	service.ErrorServices[app.Tasks[1].ID] = fmt.Errorf("Cannot register task")
	handler := NewEventHandler(service, marathon)
	body := healthStatusChangeEventForTask("/test/app.1")
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	servicesIds := service.RegisteredServicesIds()
	// then
	assert.Equal(t, 500, recorder.Code)
	assert.Equal(t, "Cannot register task\n", recorder.Body.String())
	assert.Len(t, servicesIds, 0)
}

func TestForwardHandler_NotHandleHealthStatusEventForTaskWithNotAllHeathChecksPassed(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("/test/app", 3)
	app.Tasks[1].HealthCheckResults = []tasks.HealthCheckResult{tasks.HealthCheckResult{Alive: true}, tasks.HealthCheckResult{Alive: false}}
	marathon := marathon.MarathonerStubForApps(app)
	service := consul.NewConsulStub()
	handler := NewEventHandler(service, marathon)
	body := healthStatusChangeEventForTask("/test/app.1")
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	servicesIds := service.RegisteredServicesIds()
	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "Task /test/app.1 is not healthy. Not registering\n", recorder.Body.String())
	assert.Len(t, servicesIds, 0)
}

func TestForwardHandler_NotHandleHealthStatusEventForTaskWithNoHealthCheck(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("/test/app", 1)
	app.Tasks[0].HealthCheckResults = []tasks.HealthCheckResult{}
	marathon := marathon.MarathonerStubForApps(app)
	service := consul.NewConsulStub()
	handler := NewEventHandler(service, marathon)
	body := healthStatusChangeEventForTask("/test/app.0")
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	servicesIds := service.RegisteredServicesIds()
	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "Task /test/app.0 is not healthy. Not registering\n", recorder.Body.String())
	assert.Len(t, servicesIds, 0)
}

func TestForwardHandler_NotHandleHealthStatusEventWhenTaskIsNotAlive(t *testing.T) {
	t.Parallel()
	// given
	handler := NewEventHandler(nil, nil)
	body := `{
	  "appId":"/test/app",
	  "taskId":"/test/app.1",
	  "version":"2015-12-07T09:02:48.981Z",
	  "alive":false,
	  "eventType":"health_status_changed_event",
	  "timestamp":"2015-12-07T09:33:50.069Z"
	}`
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "Task /test/app.1 is not healthy. Not registering\n", recorder.Body.String())
}

func TestForwardHandler_NotHandleHealthStatusEventWhenBodyIsInvalid(t *testing.T) {
	t.Parallel()
	// given
	handler := NewEventHandler(nil, nil)
	body := `{
	  "appId":"/test/app",
	  "taskId":"/test/app.1",
	  "version":123,
	  "alive":false,
	  "eventType":"health_status_changed_event",
	  "timestamp":"2015-12-07T09:33:50.069Z"
	}`
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "json: cannot unmarshal number into Go value of type string\n", recorder.Body.String())
}

func TestForwardHandler_HandleHealthStatusEventReturn500WhenMarathonReturnedError(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)
	handler := NewEventHandler(nil, marathon)
	body := `{
	  "appId":"unknown",
	  "taskId":"unknown.1",
	  "version":"2015-12-07T09:02:48.981Z",
	  "alive":true,
	  "eventType":"health_status_changed_event",
	  "timestamp":"2015-12-07T09:33:50.069Z"
	}`
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	// then
	assert.Equal(t, 500, recorder.Code)
	assert.Equal(t, "app not found\n", recorder.Body.String())
}

func TestForwardHandler_HandleHealthStatusEventWhenTaskIsNotInMarathon(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)
	handler := NewEventHandler(nil, marathon)
	body := healthStatusChangeEventForTask("unknown.1")
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))
	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	// then
	assert.Equal(t, 500, recorder.Code)
	assert.Equal(t, "Task unknown.1 not found\n", recorder.Body.String())
}

type BadReader struct{}

func (r BadReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("Some error")
}

func healthStatusChangeEventForTask(taskId string) string {
	return `{
	  "appId":"/test/app",
	  "taskId":"` + taskId + `",
	  "version":"2015-12-07T09:02:48.981Z",
	  "alive":true,
	  "eventType":"health_status_changed_event",
	  "timestamp":"2015-12-07T09:33:50.069Z"
	}`
}
