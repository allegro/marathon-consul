package web

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/marathon"
	. "github.com/allegro/marathon-consul/utils"
	"github.com/stretchr/testify/assert"
)

func TestWebHandler_NotHandleUnknownEventType(t *testing.T) {
	t.Parallel()

	// given
	queue := make(chan event)
	handler := newWebHandler(queue)
	stopChan := newEventHandler(0, nil, nil, queue).start()
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(`{"eventType":"test_event"}`)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	stopChan <- stopEvent{}

	// then
	assertAccepted(t, recorder)
}

func TestWebHandler_HandleRadderError(t *testing.T) {
	t.Parallel()

	// given
	handler := newWebHandler(nil)
	req, _ := http.NewRequest("POST", "/events", BadReader{})

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "Some error\n", recorder.Body.String())
}

func TestWebHandler_HandleEmptyBody(t *testing.T) {
	t.Parallel()

	// given
	handler := newWebHandler(nil)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte{}))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "unexpected end of JSON input\n", recorder.Body.String())
}

func TestWebHandler_NotHandleMalformedEventType(t *testing.T) {
	t.Parallel()

	// given
	handler := newWebHandler(nil)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(`{eventType:"test_event"}`)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "invalid character 'e' looking for beginning of object key string\n", recorder.Body.String())
}

func TestWebHandler_HandleMalformedEventType(t *testing.T) {
	t.Parallel()

	// given
	handler := newWebHandler(nil)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(`{eventType:"test_event"}`)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "invalid character 'e' looking for beginning of object key string\n", recorder.Body.String())
}

func TestWebHandler_NotHandleInvalidEventType(t *testing.T) {
	t.Parallel()

	// given
	handler := newWebHandler(nil)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(`{"eventType":[1,2]}`)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "json: cannot unmarshal array into Go value of type string\n", recorder.Body.String())
}

func TestWebHandler_HandleAppInvalidBody(t *testing.T) {
	t.Parallel()

	// given
	queue := make(chan event)
	handler := newWebHandler(queue)
	stopChan := newEventHandler(0, nil, nil, queue).start()
	body := `{"type":  "app_terminated_event", "appID": 123}`
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	stopChan <- stopEvent{}

	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Equal(t, "no event\n", recorder.Body.String())
}

func TestWebHandler_NotHandleStatusEventWithInvalidBody(t *testing.T) {
	t.Parallel()

	// given
	queue := make(chan event)
	handler := newWebHandler(queue)
	stopChan := newEventHandler(0, nil, nil, queue).start()
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
	stopChan <- stopEvent{}

	// then
	assertAccepted(t, recorder)
}

func TestWebHandler_NotHandleStatusEventAboutStartingTask(t *testing.T) {
	t.Parallel()

	// given
	services := consul.NewConsulStub()
	queue := make(chan event)
	stopChan := newEventHandler(0, services, nil, queue).start()
	handler := newWebHandler(queue)
	ignoredTaskStatuses := []string{"TASK_STAGING", "TASK_STARTING", "TASK_RUNNING", "unknown"}
	for _, taskStatus := range ignoredTaskStatuses {
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
		stopChan <- stopEvent{}

		// then
		assert.Equal(t, 202, recorder.Code)
		assert.Equal(t, "OK\n", recorder.Body.String())
		assert.Empty(t, services.RegisteredTaskIDs())
	}
}

func TestWebHandler_HandleStatusEventAboutDeadTask(t *testing.T) {
	t.Parallel()
	taskStatuses := []string{"TASK_FINISHED", "TASK_FAILED", "TASK_KILLED", "TASK_LOST"}
	for _, taskStatus := range taskStatuses {
		// given
		app := ConsulApp("/test/app", 3)
		service := consul.NewConsulStub()
		for _, task := range app.Tasks {
			service.Register(&task, app)
		}
		queue := make(chan event)
		stopChan := newEventHandler(0, service, nil, queue).start()
		handler := newWebHandler(queue)
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
		stopChan <- stopEvent{}
		taskIds := service.RegisteredTaskIDs()

		// then
		assert.Equal(t, 202, recorder.Code)
		assert.Equal(t, "OK\n", recorder.Body.String())
		assert.Len(t, taskIds, 2)
		assert.NotContains(t, taskIds, app.Tasks[1].ID)
		assert.Contains(t, taskIds, app.Tasks[0].ID)
		assert.Contains(t, taskIds, app.Tasks[2].ID)
	}
}

func TestWebHandler_HandleStatusEventAboutDeadTaskErrOnDeregistration(t *testing.T) {
	t.Parallel()

	// given
	service := consul.NewConsulStub()
	service.FailDeregisterByTaskForID("task.1")
	queue := make(chan event)
	stopChan := newEventHandler(0, service, nil, queue).start()
	handler := newWebHandler(queue)
	body := `{
	  "slaveId":"85e59460-a99e-4f16-b91f-145e0ea595bd-S0",
	  "taskId":"task.1",
	  "taskStatus":"TASK_KILLED",
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
	stopChan <- stopEvent{}

	// then
	assertAccepted(t, recorder)
	assert.Empty(t, service.RegisteredTaskIDs())
}

func TestWebHandler_NotHandleStatusEventAboutNonConsulAppsDeadTask(t *testing.T) {
	t.Parallel()

	// given
	app := NonConsulApp("/test/app", 3)
	service := consul.NewConsulStub()
	queue := make(chan event)
	stopChan := newEventHandler(0, service, nil, queue).start()
	handler := newWebHandler(queue)
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
		stopChan <- stopEvent{}

		// then
		assert.Equal(t, 202, recorder.Code)
		assert.Equal(t, "OK\n", recorder.Body.String())
	}
}

func TestWebHandler_NotHandleHealthStatusEventWhenAppHasNotConsulLabel(t *testing.T) {
	t.Parallel()

	// given
	app := NonConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)
	queue := make(chan event)
	stopChan := newEventHandler(0, nil, marathon, queue).start()
	handler := newWebHandler(queue)
	body := healthStatusChangeEventForTask("test_app.1")
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	stopChan <- stopEvent{}

	// then
	assertAccepted(t, recorder)
	assert.True(t, marathon.Interactions())
}

func TestWebHandler_HandleHealthStatusEvent(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)
	service := consul.NewConsulStub()
	handle, stop := NewHandler(Config{WorkersCount: 1}, marathon, service)
	body := healthStatusChangeEventForTask("test_app.1")
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handle(recorder, req)
	stop()
	taskIds := service.RegisteredTaskIDs()

	// then
	assertAccepted(t, recorder)
	assert.Len(t, taskIds, 1)
	assert.Contains(t, taskIds, app.Tasks[1].ID)
	assert.True(t, marathon.Interactions())
}

func TestWebHandler_HandleHealthStatusEventWithErrorsOnRegistration(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)
	service := consul.NewConsulStub()
	service.FailRegisterForID(app.Tasks[1].ID)
	handle, stop := NewHandler(Config{WorkersCount: 1}, marathon, service)
	body := healthStatusChangeEventForTask("test_app.1")
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handle(recorder, req)
	stop()

	// then
	assertAccepted(t, recorder)
	assert.Empty(t, service.RegisteredTaskIDs())
	assert.True(t, marathon.Interactions())
}

func TestWebHandler_NotHandleHealthStatusEventForTaskWithNotAllHealthChecksPassed(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	app.Tasks[1].HealthCheckResults = []apps.HealthCheckResult{{Alive: true}, {Alive: false}}
	marathon := marathon.MarathonerStubForApps(app)
	queue := make(chan event)
	stopChan := newEventHandler(0, nil, marathon, queue).start()
	handler := newWebHandler(queue)
	body := healthStatusChangeEventForTask("test_app.1")
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	stopChan <- stopEvent{}

	// then
	assertAccepted(t, recorder)
	assert.True(t, marathon.Interactions())
}

func TestWebHandler_NotHandleHealthStatusEventForTaskWithNoHealthCheck(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 1)
	app.Tasks[0].HealthCheckResults = []apps.HealthCheckResult{}
	marathon := marathon.MarathonerStubForApps(app)
	queue := make(chan event)
	stopChan := newEventHandler(0, nil, marathon, queue).start()
	handler := newWebHandler(queue)
	body := healthStatusChangeEventForTask("/test/app.0")
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	stopChan <- stopEvent{}

	// then
	assertAccepted(t, recorder)
	assert.True(t, marathon.Interactions())
}

func TestWebHandler_NotHandleHealthStatusEventWhenTaskIsNotAlive(t *testing.T) {
	t.Parallel()

	// given
	queue := make(chan event)
	stopChan := newEventHandler(0, nil, nil, queue).start()
	handler := newWebHandler(queue)
	body := `{
	  "appId":"/test/app",
	  "taskId":"test_app.1",
	  "version":"2015-12-07T09:02:48.981Z",
	  "alive":false,
	  "eventType":"health_status_changed_event",
	  "timestamp":"2015-12-07T09:33:50.069Z"
	}`
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	stopChan <- stopEvent{}

	// then
	assertAccepted(t, recorder)
}

func TestWebHandler_NotHandleHealthStatusEventWhenBodyIsInvalid(t *testing.T) {
	t.Parallel()

	// given
	queue := make(chan event)
	stopChan := newEventHandler(0, nil, nil, queue).start()
	handler := newWebHandler(queue)
	body := `{
	  "appId":"/test/app",
	  "taskId":"test_app.1",
	  "version":123,
	  "alive":false,
	  "eventType":"health_status_changed_event",
	  "timestamp":"2015-12-07T09:33:50.069Z"
	}`
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	stopChan <- stopEvent{}

	// then
	assertAccepted(t, recorder)
}

func TestWebHandler_HandleHealthStatusEventReturn202WhenMarathonReturnedError(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)
	queue := make(chan event)
	stopChan := newEventHandler(0, nil, marathon, queue).start()
	handler := newWebHandler(queue)
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
	stopChan <- stopEvent{}

	// then
	assertAccepted(t, recorder)
	assert.True(t, marathon.Interactions())
}

func TestWebHandler_HandleHealthStatusEventWhenTaskIsNotInMarathon(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)
	queue := make(chan event)
	stopChan := newEventHandler(0, nil, marathon, queue).start()
	handler := newWebHandler(queue)
	body := healthStatusChangeEventForTask("unknown.1")
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	stopChan <- stopEvent{}

	// then
	assertAccepted(t, recorder)
	assert.True(t, marathon.Interactions())
}

func newConsulStubWithApplicationsTasksRegistered(applications ...*apps.App) *consul.Stub {
	service := consul.NewConsulStub()
	for _, app := range applications {
		for _, task := range app.Tasks {
			service.Register(&task, app)
		}
	}
	return service
}

type BadReader struct{}

func (r BadReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("Some error")
}

func healthStatusChangeEventForTask(taskID string) string {
	return `{
	  "appId":"/test/app",
	  "taskId":"` + taskID + `",
	  "version":"2015-12-07T09:02:48.981Z",
	  "alive":true,
	  "eventType":"health_status_changed_event",
	  "timestamp":"2015-12-07T09:33:50.069Z"
	}`
}

func assertAccepted(t *testing.T, recorder *httptest.ResponseRecorder) {
	assert.Equal(t, 202, recorder.Code)
	assert.Equal(t, "OK\n", recorder.Body.String())
}
