package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/events"
	"github.com/allegro/marathon-consul/marathon"
	. "github.com/allegro/marathon-consul/utils"
	"github.com/stretchr/testify/assert"
)

func TestWebHandler_NotHandleUnknownEventType(t *testing.T) {
	t.Parallel()

	// given
	queue := make(chan event)
	handler := newWebHandler(queue)
	stopChan := newEventHandler(0, nil, nil, queue).Start()
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

	if runtime.Version() < "go1.8" && !strings.HasPrefix(runtime.Version(), "devel") {
		assert.Equal(t, "json: cannot unmarshal array into Go value of type string\n", recorder.Body.String())
	} else {
		assert.Equal(t, "json: cannot unmarshal array into Go struct field BaseEvent.eventType of type string\n", recorder.Body.String())
	}
}

func TestWebHandler_HandleAppInvalidBody(t *testing.T) {
	t.Parallel()

	// given
	queue := make(chan event)
	handler := newWebHandler(queue)
	stopChan := newEventHandler(0, nil, nil, queue).Start()
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

func TestWebHandler_HandleDeploymentInfoWithStopApplicationAction(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	handle, stop := NewHandler(Config{WorkersCount: 1}, marathon, service)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(app))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handle(recorder, req)
	stop()

	// then
	assertAccepted(t, recorder)
	assert.Empty(t, service.RegisteredServicesIds())
	assert.False(t, marathon.Interactions())
}

func TestWebHandler_handleDeploymentInfoWithStopApplicationForOneApp(t *testing.T) {
	t.Parallel()

	// given
	green := ConsulApp("/test/app.green", 3)
	green.Labels[apps.MARATHON_CONSUL_LABEL] = "app"
	blue := ConsulApp("/test/app.blue", 2)
	blue.Labels[apps.MARATHON_CONSUL_LABEL] = "app"
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(green, blue)
	assert.Len(t, service.RegisteredServicesIds(), 5)
	handle, stop := NewHandler(Config{WorkersCount: 1}, marathon, service)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(green))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handle(recorder, req)
	stop()

	// then
	assertAccepted(t, recorder)
	assert.Len(t, service.RegisteredServicesIds(), 2)
	assert.False(t, marathon.Interactions())
}

func TestWebHandler_HandleDeploymentInfoWithStopApplicationActionForMultipleApps(t *testing.T) {
	t.Parallel()

	// given
	app1 := ConsulApp("/test/app", 3)
	app2 := ConsulApp("/test/otherapp", 2)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app1, app2)
	assert.Len(t, service.RegisteredServicesIds(), 5)
	handle, stop := NewHandler(Config{WorkersCount: 1}, marathon, service)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(app1, app2))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handle(recorder, req)
	stop()

	// then
	assertAccepted(t, recorder)
	assert.Len(t, service.RegisteredServicesIds(), 0)
	assert.False(t, marathon.Interactions())
}

func TestWebHandler_HandleDeploymentInfoWithStopApplicationActionForMultipleAppsAndProblemsDeregisteringOne(t *testing.T) {
	t.Parallel()

	// given
	app1 := ConsulApp("/test/app", 3)
	app2 := ConsulApp("/test/otherapp", 2)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app1, app2)
	service.ErrorServices["test_app.1"] = fmt.Errorf("Cannot deregister service")
	assert.Len(t, service.RegisteredServicesIds(), 5)
	handle, stop := NewHandler(Config{WorkersCount: 1}, marathon, service)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(app1, app2))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handle(recorder, req)
	stop()

	// then
	assertAccepted(t, recorder)
	assert.Len(t, service.RegisteredServicesIds(), 1)
	assert.False(t, marathon.Interactions())
}

func TestWebHandler_HandleDeploymentInfoWithStopApplicationActionForMultipleAppsAndProblemsGettingServicesForOne(t *testing.T) {
	t.Parallel()

	// given
	app1 := ConsulApp("/test/app", 3)
	app2 := ConsulApp("/test/otherapp", 2)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app1, app2)
	service.ErrorGetServices["test.app"] = fmt.Errorf("Something went terribly wrong!")
	assert.Len(t, service.RegisteredServicesIds(), 5)
	handle, stop := NewHandler(Config{WorkersCount: 1}, marathon, service)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(app1, app2))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handle(recorder, req)
	stop()

	// then
	assertAccepted(t, recorder)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	assert.False(t, marathon.Interactions())
}

func TestWebHandler_HandleDeploymentInfoWithStopApplicationActionWithNoServicesRegistered(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps()
	service := consul.NewConsulStub()
	assert.Len(t, service.RegisteredServicesIds(), 0)
	handle, stop := NewHandler(Config{WorkersCount: 1}, marathon, service)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(app))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handle(recorder, req)
	stop()

	// then
	assertAccepted(t, recorder)
	assert.Len(t, service.RegisteredServicesIds(), 0)
	assert.False(t, marathon.Interactions())
}

func TestWebHandler_HandleDeploymentInfoWithInvalidBody(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	handle, stop := NewHandler(Config{WorkersCount: 1}, marathon, service)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(`{"eventType":"deployment_info", "Plan": 123}`)))

	// when
	recorder := httptest.NewRecorder()
	handle(recorder, req)
	stop()

	// then
	assertAccepted(t, recorder)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	assert.False(t, marathon.Interactions())
}

func TestWebHandler_HandleDeploymentInfoWithStopApplicationActionForNonConsulApp(t *testing.T) {
	t.Parallel()

	// given
	app := NonConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	handle, stop := NewHandler(Config{WorkersCount: 1}, marathon, service)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(app))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handle(recorder, req)
	stop()

	// then
	assertAccepted(t, recorder)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	assert.False(t, marathon.Interactions())
}

func TestWebHandler_HandleDeploymentInfoWithStopApplicationActionForCustomServiceName(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	app.Labels["consul"] = "someCustomServiceName"
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	handle, stop := NewHandler(Config{WorkersCount: 1}, marathon, service)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(app))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handle(recorder, req)
	stop()

	// then
	assertAccepted(t, recorder)
	assert.Len(t, service.RegisteredServicesIds(), 0)
	assert.False(t, marathon.Interactions())
}

func TestWebHandler_NotHandleDeploymentInfoWithScaleApplicationAction(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	handle, stop := NewHandler(Config{WorkersCount: 1}, marathon, service)
	deploymentInfo := deploymentInfoWithStopApplicationActionForApps(app)
	deploymentInfo.CurrentStep.Actions[0].Type = "ScaleApplication"
	body, _ := json.Marshal(deploymentInfo)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handle(recorder, req)
	stop()

	// then
	assertAccepted(t, recorder)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	assert.False(t, marathon.Interactions())
}

func TestWebHandler_HandleDeploymentInfoWithStopApplicationActionAndProblemsDeregistering(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app)
	service.ErrorServices["test_app.1"] = fmt.Errorf("Cannot deregister service")
	assert.Len(t, service.RegisteredServicesIds(), 3)
	handle, stop := NewHandler(Config{WorkersCount: 1}, marathon, service)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(app))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handle(recorder, req)
	stop()

	// then
	assertAccepted(t, recorder)
	assert.Len(t, service.RegisteredServicesIds(), 1)
	assert.False(t, marathon.Interactions())
}

func deploymentInfoWithStopApplicationActionForApps(applications ...*apps.App) *events.DeploymentEvent {

	deploymentInfo := &events.DeploymentEvent{
		Type: "deployment_info",
		Plan: &events.Plan{
			Original: &events.Deployments{
				Apps: []*apps.App{},
			},
		},
		CurrentStep: &events.CurrentStep{
			Actions: []*events.Action{},
		},
	}
	for _, app := range applications {
		deploymentInfo.Plan.Original.Apps = append(deploymentInfo.Plan.Original.Apps, app)
		deploymentInfo.CurrentStep.Actions = append(deploymentInfo.CurrentStep.Actions, &events.Action{AppId: app.ID, Type: "StopApplication"})
	}
	return deploymentInfo
}

func TestWebHandler_HandleDeploymentStepSuccessWithRestartAndRenameApplicationAction(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	handle, stop := NewHandler(Config{WorkersCount: 1}, marathon, service)
	body, _ := json.Marshal(deploymentStepSuccessWithRestartAndRenameApplicationActionForApps(app))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handle(recorder, req)
	stop()

	// then
	assertAccepted(t, recorder)
	assert.Len(t, service.RegisteredServicesIds(), 0)
	assert.False(t, marathon.Interactions())
}

func TestWebHandler_HandleDeploymentStepSuccessWithInvalidBody(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	handle, stop := NewHandler(Config{WorkersCount: 1}, marathon, service)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(`{"eventType":"deployment_step_success", "Plan": 123}`)))

	// when
	recorder := httptest.NewRecorder()
	handle(recorder, req)
	stop()

	// then
	assertAccepted(t, recorder)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	assert.False(t, marathon.Interactions())
}

func TestWebHandler_HandleDeploymentStepSuccessWithRestartApplicationActionForMultipleAppsAndProblemsDeregisteringOne(t *testing.T) {
	t.Parallel()

	// given
	app1 := ConsulApp("/test/app", 3)
	app2 := ConsulApp("/test/otherapp", 2)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app1, app2)
	service.ErrorServices["test_app.1"] = fmt.Errorf("Cannot deregister service")
	assert.Len(t, service.RegisteredServicesIds(), 5)
	handle, stop := NewHandler(Config{WorkersCount: 1}, marathon, service)
	body, _ := json.Marshal(deploymentStepSuccessWithRestartAndRenameApplicationActionForApps(app1, app2))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handle(recorder, req)
	stop()

	// then
	assertAccepted(t, recorder)
	assert.Len(t, service.RegisteredServicesIds(), 1)
	assert.False(t, marathon.Interactions())
}

func deploymentStepSuccessWithRestartAndRenameApplicationActionForApps(applications ...*apps.App) *events.DeploymentEvent {
	deploymentInfo := &events.DeploymentEvent{
		Type: "deployment_step_success",
		Plan: &events.Plan{
			Original: &events.Deployments{
				Apps: []*apps.App{},
			},
			Target: &events.Deployments{
				Apps: []*apps.App{},
			},
		},
		CurrentStep: &events.CurrentStep{
			Actions: []*events.Action{},
		},
	}
	for _, app := range applications {
		deploymentInfo.Plan.Original.Apps = append(deploymentInfo.Plan.Original.Apps, app)
		targetApp := &apps.App{ID: app.ID, Labels: map[string]string{}}
		if name, ok := app.Labels["consul"]; ok {
			targetApp.Labels["consul"] = fmt.Sprintf("New%s", name)
		}
		deploymentInfo.Plan.Target.Apps = append(deploymentInfo.Plan.Target.Apps, targetApp)
		deploymentInfo.CurrentStep.Actions = append(deploymentInfo.CurrentStep.Actions, &events.Action{AppId: app.ID, Type: "RestartApplication"})
	}
	return deploymentInfo
}

func TestWebHandler_NotHandleStatusEventWithInvalidBody(t *testing.T) {
	t.Parallel()

	// given
	queue := make(chan event)
	handler := newWebHandler(queue)
	stopChan := newEventHandler(0, nil, nil, queue).Start()
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
	stopChan := newEventHandler(0, services, nil, queue).Start()
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
		assert.Empty(t, services.RegisteredServicesIds())
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
		stopChan := newEventHandler(0, service, nil, queue).Start()
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
		servicesIds := service.RegisteredServicesIds()

		// then
		assert.Equal(t, 202, recorder.Code)
		assert.Equal(t, "OK\n", recorder.Body.String())
		assert.Len(t, servicesIds, 2)
		assert.NotContains(t, servicesIds, app.Tasks[1].ID)
		assert.Contains(t, servicesIds, app.Tasks[0].ID.String())
		assert.Contains(t, servicesIds, app.Tasks[2].ID.String())
	}
}

func TestWebHandler_HandleEventAboutUnhealthyKilledTask(t *testing.T) {
	t.Parallel()
	// given
	app := ConsulApp("/test/app", 3)
	service := consul.NewConsulStub()
	for _, task := range app.Tasks {
		service.Register(&task, app)
	}
	queue := make(chan event)
	stopChan := newEventHandler(0, service, nil, queue).Start()
	handler := newWebHandler(queue)
	body := `{
	  "appId": "/test/app",
	  "taskId": "` + app.Tasks[1].ID.String() + `",
	  "version": "2016-03-16T13:05:00.590Z",
	  "reason": "500 Internal Server Error",
	  "host": "localhost",
	  "slaveId": "4fb620fa-ba8d-4eb0-8ae3-f2912aaf015c-S0",
	  "eventType": "unhealthy_task_kill_event",
	  "timestamp": "2016-03-21T09:15:10.764Z"
	}`
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	stopChan <- stopEvent{}
	servicesIds := service.RegisteredServicesIds()

	// then
	assert.Equal(t, 202, recorder.Code)
	assert.Equal(t, "OK\n", recorder.Body.String())
	assert.Len(t, servicesIds, 2)
	assert.NotContains(t, servicesIds, app.Tasks[1].ID)
	assert.Contains(t, servicesIds, app.Tasks[0].ID.String())
	assert.Contains(t, servicesIds, app.Tasks[2].ID.String())
}

func TestWebHandler_NotHandleEventAboutUnhealthyKilledTaskWithInvalidBody(t *testing.T) {
	t.Parallel()

	// given
	queue := make(chan event)
	handler := newWebHandler(queue)
	stopChan := newEventHandler(0, nil, nil, queue).Start()
	body := `{
	  "appId": "/test/app",
	  "taskId": "test.app.1",
	  "version": "2016-03-16T13:05:00.590Z",
	  "reason": "500 Internal Server Error",
	  "host": "localhost",
	  "ports": 31372,
	  "slaveId": "4fb620fa-ba8d-4eb0-8ae3-f2912aaf015c-S0",
	  "eventType": "unhealthy_task_kill_event",
	  "timestamp": "2016-03-21T09:15:10.764Z"
	}`
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	stopChan <- stopEvent{}

	// then
	assertAccepted(t, recorder)
}

func TestWebHandler_HandleStatusEventAboutDeadTaskErrOnDeregistration(t *testing.T) {
	t.Parallel()

	// given
	service := consul.NewConsulStub()
	service.ErrorServices[apps.TaskId("task.1")] = fmt.Errorf("Cannot deregister task")
	queue := make(chan event)
	stopChan := newEventHandler(0, service, nil, queue).Start()
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
	assert.Empty(t, service.RegisteredServicesIds())
}

func TestWebHandler_NotHandleStatusEventAboutNonConsulAppsDeadTask(t *testing.T) {
	t.Parallel()

	// given
	app := NonConsulApp("/test/app", 3)
	service := consul.NewConsulStub()
	queue := make(chan event)
	stopChan := newEventHandler(0, service, nil, queue).Start()
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
	stopChan := newEventHandler(0, nil, marathon, queue).Start()
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
	servicesIds := service.RegisteredServicesIds()

	// then
	assertAccepted(t, recorder)
	assert.Len(t, servicesIds, 1)
	assert.Contains(t, servicesIds, app.Tasks[1].ID.String())
	assert.True(t, marathon.Interactions())
}

func TestWebHandler_HandleHealthStatusEventWithErrorsOnRegistration(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)
	service := consul.NewConsulStub()
	service.ErrorServices[app.Tasks[1].ID] = fmt.Errorf("Cannot register task")
	handle, stop := NewHandler(Config{WorkersCount: 1}, marathon, service)
	body := healthStatusChangeEventForTask("test_app.1")
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handle(recorder, req)
	stop()

	// then
	assertAccepted(t, recorder)
	assert.Empty(t, service.RegisteredServicesIds())
	assert.True(t, marathon.Interactions())
}

func TestWebHandler_NotHandleHealthStatusEventForTaskWithNotAllHealthChecksPassed(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	app.Tasks[1].HealthCheckResults = []apps.HealthCheckResult{{Alive: true}, {Alive: false}}
	marathon := marathon.MarathonerStubForApps(app)
	queue := make(chan event)
	stopChan := newEventHandler(0, nil, marathon, queue).Start()
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
	stopChan := newEventHandler(0, nil, marathon, queue).Start()
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
	stopChan := newEventHandler(0, nil, nil, queue).Start()
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
	stopChan := newEventHandler(0, nil, nil, queue).Start()
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
	stopChan := newEventHandler(0, nil, marathon, queue).Start()
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
	stopChan := newEventHandler(0, nil, marathon, queue).Start()
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

func newConsulStubWithApplicationsTasksRegistered(applications ...*apps.App) *consul.ConsulStub {
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

func assertAccepted(t *testing.T, recorder *httptest.ResponseRecorder) {
	assert.Equal(t, 202, recorder.Code)
	assert.Equal(t, "OK\n", recorder.Body.String())
}
