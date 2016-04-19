package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/events"
	"github.com/allegro/marathon-consul/marathon"
	. "github.com/allegro/marathon-consul/utils"
	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, 200, recorder.Code)
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

func TestForwardHandler_HandleDeploymentInfoWithStopApplicationAction(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	handler := NewEventHandler(service, marathon)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(app))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 200, recorder.Code)
	assert.Len(t, service.RegisteredServicesIds(), 0)
}

func TestForwardHandler_HandleDeploymentInfoWithStopApplicationActionForMultipleApps(t *testing.T) {
	t.Parallel()

	// given
	app1 := ConsulApp("/test/app", 3)
	app2 := ConsulApp("/test/otherapp", 2)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app1, app2)
	assert.Len(t, service.RegisteredServicesIds(), 5)
	handler := NewEventHandler(service, marathon)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(app1, app2))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 200, recorder.Code)
	assert.Len(t, service.RegisteredServicesIds(), 0)
}

func TestForwardHandler_HandleDeploymentInfoWithStopApplicationForOneApp(t *testing.T) {
	t.Parallel()

	// given
	green := ConsulApp("/test/app.green", 3)
	green.Labels[apps.MARATHON_CONSUL_LABEL] = "app"
	blue := ConsulApp("/test/app.blue", 2)
	blue.Labels[apps.MARATHON_CONSUL_LABEL] = "app"
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(green, blue)
	assert.Len(t, service.RegisteredServicesIds(), 5)
	handler := NewEventHandler(service, marathon)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(green))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 200, recorder.Code)
	assert.Len(t, service.RegisteredServicesIds(), 2)
}

func TestForwardHandler_HandleDeploymentInfoWithStopApplicationActionForMultipleAppsAndProblemsDeregisteringOne(t *testing.T) {
	t.Parallel()

	// given
	app1 := ConsulApp("/test/app", 3)
	app2 := ConsulApp("/test/otherapp", 2)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app1, app2)
	service.ErrorServices["test_app.1"] = fmt.Errorf("Cannot deregister service")
	assert.Len(t, service.RegisteredServicesIds(), 5)
	handler := NewEventHandler(service, marathon)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(app1, app2))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 500, recorder.Code)
	assert.Len(t, service.RegisteredServicesIds(), 1)
}

func TestForwardHandler_HandleDeploymentInfoWithStopApplicationActionForMultipleAppsAndProblemsGettingServicesForOne(t *testing.T) {
	t.Parallel()

	// given
	app1 := ConsulApp("/test/app", 3)
	app2 := ConsulApp("/test/otherapp", 2)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app1, app2)
	service.ErrorGetServices["test.app"] = fmt.Errorf("Something went terribly wrong!")
	assert.Len(t, service.RegisteredServicesIds(), 5)
	handler := NewEventHandler(service, marathon)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(app1, app2))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 500, recorder.Code)
	assert.Len(t, service.RegisteredServicesIds(), 3)
}

func TestForwardHandler_HandleDeploymentInfoWithStopApplicationActionWithNoServicesRegistered(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps()
	service := consul.NewConsulStub()
	assert.Len(t, service.RegisteredServicesIds(), 0)
	handler := NewEventHandler(service, marathon)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(app))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 200, recorder.Code)
	assert.Len(t, service.RegisteredServicesIds(), 0)
}

func TestForwardHandler_HandleDeploymentInfoWithInvalidBody(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	handler := NewEventHandler(service, marathon)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(`{"eventType":"deployment_info", "Plan": 123}`)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Len(t, service.RegisteredServicesIds(), 3)
}

func TestForwardHandler_HandleDeploymentInfoWithStopApplicationActionForNonConsulApp(t *testing.T) {
	t.Parallel()

	// given
	app := NonConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	handler := NewEventHandler(service, marathon)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(app))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 200, recorder.Code)
	assert.Len(t, service.RegisteredServicesIds(), 3)
}

func TestForwardHandler_HandleDeploymentInfoWithStopApplicationActionForCustomServiceName(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	app.Labels["consul"] = "someCustomServiceName"
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	handler := NewEventHandler(service, marathon)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(app))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 200, recorder.Code)
	assert.Len(t, service.RegisteredServicesIds(), 0)
}

func TestForwardHandler_NotHandleDeploymentInfoWithScaleApplicationAction(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	handler := NewEventHandler(service, marathon)
	deploymentInfo := deploymentInfoWithStopApplicationActionForApps(app)
	deploymentInfo.CurrentStep.Actions[0].Type = "ScaleApplication"
	body, _ := json.Marshal(deploymentInfo)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 200, recorder.Code)
	assert.Len(t, service.RegisteredServicesIds(), 3)
}

func TestForwardHandler_HandleDeploymentInfoWithStopApplicationActionAndProblemsDeregistering(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app)
	service.ErrorServices["test_app.1"] = fmt.Errorf("Cannot deregister service")
	assert.Len(t, service.RegisteredServicesIds(), 3)
	handler := NewEventHandler(service, marathon)
	body, _ := json.Marshal(deploymentInfoWithStopApplicationActionForApps(app))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 500, recorder.Code)
	assert.Equal(t, "1 errors occured handling service deregistration\n0: Cannot deregister service\n", recorder.Body.String())
	assert.Len(t, service.RegisteredServicesIds(), 1)
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

func TestForwardHandler_HandleDeploymentStepSuccessWithRestartAndRenameApplicationAction(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	handler := NewEventHandler(service, marathon)
	body, _ := json.Marshal(deploymentStepSuccessWithRestartAndRenameApplicationActionForApps(app))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 200, recorder.Code)
	assert.Len(t, service.RegisteredServicesIds(), 0)
}

func TestForwardHandler_HandleDeploymentStepSuccessWithInvalidBody(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app)
	assert.Len(t, service.RegisteredServicesIds(), 3)
	handler := NewEventHandler(service, marathon)
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(`{"eventType":"deployment_step_success", "Plan": 123}`)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 400, recorder.Code)
	assert.Len(t, service.RegisteredServicesIds(), 3)
}

func TestForwardHandler_HandleDeploymentStepSuccessWithRestartApplicationActionForMultipleAppsAndProblemsDeregisteringOne(t *testing.T) {
	t.Parallel()

	// given
	app1 := ConsulApp("/test/app", 3)
	app2 := ConsulApp("/test/otherapp", 2)
	marathon := marathon.MarathonerStubForApps()
	service := newConsulStubWithApplicationsTasksRegistered(app1, app2)
	service.ErrorServices["test_app.1"] = fmt.Errorf("Cannot deregister service")
	assert.Len(t, service.RegisteredServicesIds(), 5)
	handler := NewEventHandler(service, marathon)
	body, _ := json.Marshal(deploymentStepSuccessWithRestartAndRenameApplicationActionForApps(app1, app2))
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)

	// then
	assert.Equal(t, 500, recorder.Code)
	assert.Equal(t, "1 errors occured handling service deregistration\n0: Cannot deregister service\n", recorder.Body.String())
	assert.Len(t, service.RegisteredServicesIds(), 1)
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
	services := consul.NewConsulStub()
	handler := NewEventHandler(services, nil)
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

		// then
		assert.Equal(t, 200, recorder.Code)
		assert.Len(t, services.RegisteredServicesIds(), 0)
	}
}

func TestForwardHandler_HandleStatusEventAboutDeadTask(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	service := consul.NewConsulStub()
	for _, task := range app.Tasks {
		service.Register(&task, app)
	}
	handler := NewEventHandler(service, nil)
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

func TestForwardHandler_NotHandleStatusEventAboutNonConsulAppsDeadTask(t *testing.T) {
	t.Parallel()

	// given
	app := NonConsulApp("/test/app", 3)
	service := consul.NewConsulStub()
	handler := NewEventHandler(service, nil)
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

		// then
		assert.Equal(t, 200, recorder.Code)
	}
}

func TestForwardHandler_NotHandleHealthStatusEventWhenAppHasNotConsulLabel(t *testing.T) {
	t.Parallel()

	// given
	app := NonConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)
	service := consul.NewConsulStub()
	handler := NewEventHandler(service, marathon)
	body := healthStatusChangeEventForTask("test_app.1")
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	servicesIds := service.RegisteredServicesIds()

	// then
	assert.Equal(t, 200, recorder.Code)
	assert.Len(t, servicesIds, 0)
}

func TestForwardHandler_HandleHealthStatusEvent(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)
	service := consul.NewConsulStub()
	handler := NewEventHandler(service, marathon)
	body := healthStatusChangeEventForTask("test_app.1")
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
	body := healthStatusChangeEventForTask("test_app.1")
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

func TestForwardHandler_NotHandleHealthStatusEventForTaskWithNotAllHealthChecksPassed(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	app.Tasks[1].HealthCheckResults = []apps.HealthCheckResult{{Alive: true}, {Alive: false}}
	marathon := marathon.MarathonerStubForApps(app)
	service := consul.NewConsulStub()
	handler := NewEventHandler(service, marathon)
	body := healthStatusChangeEventForTask("test_app.1")
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	servicesIds := service.RegisteredServicesIds()

	// then
	assert.Equal(t, 200, recorder.Code)
	assert.Len(t, servicesIds, 0)
}

func TestForwardHandler_NotHandleHealthStatusEventForTaskWithNoHealthCheck(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 1)
	app.Tasks[0].HealthCheckResults = []apps.HealthCheckResult{}
	marathon := marathon.MarathonerStubForApps(app)
	service := consul.NewConsulStub()
	handler := NewEventHandler(service, marathon)
	body := healthStatusChangeEventForTask("test_app.0")
	req, _ := http.NewRequest("POST", "/events", bytes.NewBuffer([]byte(body)))

	// when
	recorder := httptest.NewRecorder()
	handler.Handle(recorder, req)
	servicesIds := service.RegisteredServicesIds()

	// then
	assert.Equal(t, 200, recorder.Code)
	assert.Len(t, servicesIds, 0)
}

func TestForwardHandler_NotHandleHealthStatusEventWhenTaskIsNotAlive(t *testing.T) {
	t.Parallel()

	// given
	services := consul.NewConsulStub()
	handler := NewEventHandler(services, nil)
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

	// then
	assert.Equal(t, 200, recorder.Code)
	assert.Len(t, services.RegisteredServicesIds(), 0)
}

func TestForwardHandler_NotHandleHealthStatusEventWhenBodyIsInvalid(t *testing.T) {
	t.Parallel()

	// given
	handler := NewEventHandler(nil, nil)
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
	  "appId":"/unknown",
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
