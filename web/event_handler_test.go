package web

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/allegro/marathon-consul/apps"
	"github.com/allegro/marathon-consul/consul"
	"github.com/allegro/marathon-consul/marathon"
	"github.com/allegro/marathon-consul/service"
	. "github.com/allegro/marathon-consul/utils"
	"github.com/stretchr/testify/assert"
)

type handlerStubs struct {
	serviceRegistry service.ServiceRegistry
	marathon        marathon.Marathoner
}

// Creates eventHandler and returns nonbuffered event queue that has to be used to send events to handler and
// function that can be used as a synchronization point to wait until previous event has been processed.
// Under the hood synchronization function simply sends a stop signal to the handlers stopChan.
func testEventHandler(stubs handlerStubs) (chan<- event, func()) {
	queue := make(chan event)
	awaitChan := newEventHandler(0, stubs.serviceRegistry, stubs.marathon, queue).start()

	return queue, func() { awaitChan <- stopEvent{} }
}

func TestEventHandler_NotHandleStatusEventWithInvalidBody(t *testing.T) {
	t.Parallel()

	// given
	serviceRegistry := consul.NewConsulStub()
	queue, awaitFunc := testEventHandler(handlerStubs{serviceRegistry: serviceRegistry})

	body := []byte(`{
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
	}`)

	// when
	queue <- event{eventType: "status_update_event", timestamp: time.Now(), body: body}
	awaitFunc()

	// then
	assert.Empty(t, serviceRegistry.RegisteredTaskIDs("test_app"))
}

func TestEventHandler_NotHandleStatusEventAboutStartingTask(t *testing.T) {
	t.Parallel()

	// given
	serviceRegistry := consul.NewConsulStub()
	queue, awaitFunc := testEventHandler(handlerStubs{serviceRegistry: serviceRegistry})

	ignoredTaskStatuses := []string{"TASK_STAGING", "TASK_STARTING", "TASK_RUNNING", "unknown"}
	for _, taskStatus := range ignoredTaskStatuses {
		body := []byte(`{
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
		}`)

		// when
		queue <- event{eventType: "status_update_event", timestamp: time.Now(), body: body}
		awaitFunc()

		// then
		assert.Empty(t, serviceRegistry.RegisteredTaskIDs("test_app"))
	}
}

func TestEventHandler_HandleStatusEventAboutDeadTask(t *testing.T) {
	t.Parallel()

	// given
	serviceRegistry := consul.NewConsulStub()

	queue, awaitFunc := testEventHandler(handlerStubs{serviceRegistry: serviceRegistry})

	taskStatuses := []string{"TASK_FINISHED", "TASK_FAILED", "TASK_KILLED", "TASK_LOST"}
	for index, taskStatus := range taskStatuses {
		// given
		appId := "/test/app" + strconv.Itoa(index)
		serviceName := "test.app" + strconv.Itoa(index)

		app := ConsulApp(appId, 3)
		for _, task := range app.Tasks {
			serviceRegistry.Register(&task, app)
		}

		body := []byte(`{
		  "slaveId":"85e59460-a99e-4f16-b91f-145e0ea595bd-S0",
		  "taskId":"` + app.Tasks[1].ID.String() + `",
		  "taskStatus":"` + taskStatus + `",
		  "message":"Command terminated with signal Terminated",
		  "appId":"` + appId + `",
		  "host":"localhost",
		  "ports":[
			31372
		  ],
		  "version":"2015-12-07T09:02:48.981Z",
		  "eventType":"status_update_event",
		  "timestamp":"2015-12-07T09:33:40.898Z"
		}`)

		// when
		queue <- event{eventType: "status_update_event", timestamp: time.Now(), body: body}
		awaitFunc()

		// then
		taskIds := serviceRegistry.RegisteredTaskIDs(serviceName)
		assert.Len(t, taskIds, 2)
		assert.NotContains(t, taskIds, app.Tasks[1].ID)
		assert.Contains(t, taskIds, app.Tasks[0].ID)
		assert.Contains(t, taskIds, app.Tasks[2].ID)
	}
}

func TestEventHandler_HandleStatusEventAboutDeadTaskErrOnDeregistration(t *testing.T) {
	t.Parallel()

	// given
	serviceRegistry := consul.NewConsulStub()
	serviceRegistry.FailDeregisterByTaskForID("task.1")

	queue, awaitFunc := testEventHandler(handlerStubs{serviceRegistry: serviceRegistry})

	body := []byte(`{
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
	}`)

	// when
	queue <- event{eventType: "status_update_event", timestamp: time.Now(), body: body}
	awaitFunc()

	// then
	assert.Empty(t, serviceRegistry.RegisteredTaskIDs("test_app"))
}

func TestEventHandler_NotHandleStatusEventAboutNonConsulAppsDeadTask(t *testing.T) {
	t.Parallel()

	// given
	app := NonConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)
	serviceRegistry := consul.NewConsulStub()

	queue, awaitFunc := testEventHandler(handlerStubs{serviceRegistry: serviceRegistry, marathon: marathon})

	taskStatuses := []string{"TASK_FINISHED", "TASK_FAILED", "TASK_KILLED", "TASK_LOST"}
	for _, taskStatus := range taskStatuses {
		body := []byte(`{
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
		}`)

		// when
		queue <- event{eventType: "status_update_event", timestamp: time.Now(), body: body}
		awaitFunc()

		// then
		assert.False(t, marathon.Interactions())
	}
}

func TestEventHandler_NotHandleHealthStatusEventWhenAppHasNotConsulLabel(t *testing.T) {
	t.Parallel()

	// given
	app := NonConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)

	queue, awaitFunc := testEventHandler(handlerStubs{marathon: marathon})

	body := healthStatusChangeEventForTask("test_app.1")

	// when
	queue <- event{eventType: "health_status_changed_event", timestamp: time.Now(), body: body}
	awaitFunc()

	// then
	assert.True(t, marathon.Interactions())
}

func TestEventHandler_HandleHealthStatusEvent(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)

	marathon := marathon.MarathonerStubForApps(app)
	serviceRegistry := consul.NewConsulStub()

	queue, awaitFunc := testEventHandler(handlerStubs{serviceRegistry: serviceRegistry, marathon: marathon})
	body := healthStatusChangeEventForTask("test_app.1")

	// when
	queue <- event{eventType: "health_status_changed_event", timestamp: time.Now(), body: body}
	awaitFunc()

	// then
	taskIds := serviceRegistry.RegisteredTaskIDs("test.app")
	assert.Len(t, taskIds, 1)
	assert.Contains(t, taskIds, app.Tasks[1].ID)
	assert.True(t, marathon.Interactions())
}

func TestEventHandler_HandleHealthStatusEventWithErrorsOnRegistration(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)

	marathon := marathon.MarathonerStubForApps(app)
	serviceRegistry := consul.NewConsulStub()
	serviceRegistry.FailRegisterForID(app.Tasks[1].ID)

	queue, awaitFunc := testEventHandler(handlerStubs{serviceRegistry: serviceRegistry, marathon: marathon})
	body := healthStatusChangeEventForTask("test_app.1")

	// when
	queue <- event{eventType: "health_status_changed_event", timestamp: time.Now(), body: body}
	awaitFunc()

	// then
	assert.Empty(t, serviceRegistry.RegisteredTaskIDs("test.app"))
	assert.True(t, marathon.Interactions())
}

func TestEventHandler_NotHandleHealthStatusEventForTaskWithNotAllHealthChecksPassed(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	app.Tasks[1].HealthCheckResults = []apps.HealthCheckResult{{Alive: true}, {Alive: false}}
	marathon := marathon.MarathonerStubForApps(app)

	queue, awaitFunc := testEventHandler(handlerStubs{marathon: marathon})
	body := healthStatusChangeEventForTask("test_app.1")

	// when
	queue <- event{eventType: "health_status_changed_event", timestamp: time.Now(), body: body}
	awaitFunc()

	// then
	assert.True(t, marathon.Interactions())
}

func TestEventHandler_NotHandleHealthStatusEventForTaskWithNoHealthCheck(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 1)
	app.Tasks[0].HealthCheckResults = []apps.HealthCheckResult{}
	marathon := marathon.MarathonerStubForApps(app)

	queue, awaitFunc := testEventHandler(handlerStubs{marathon: marathon})

	body := healthStatusChangeEventForTask("/test/app.0")

	// when
	queue <- event{eventType: "health_status_changed_event", timestamp: time.Now(), body: body}
	awaitFunc()

	// then
	assert.True(t, marathon.Interactions())
}

func TestEventHandler_NotHandleHealthStatusEventWhenTaskIsNotAlive(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 1)
	marathon := marathon.MarathonerStubForApps(app)

	queue, awaitFunc := testEventHandler(handlerStubs{marathon: marathon})

	body := []byte(`{
	  "appId":"/test/app",
	  "taskId":"test_app.1",
	  "version":"2015-12-07T09:02:48.981Z",
	  "alive":false,
	  "eventType":"health_status_changed_event",
	  "timestamp":"2015-12-07T09:33:50.069Z"
	}`)

	// when
	queue <- event{eventType: "health_status_changed_event", timestamp: time.Now(), body: body}
	awaitFunc()

	// then
	assert.False(t, marathon.Interactions())
}

func TestEventHandler_NotHandleHealthStatusEventWhenBodyIsInvalid(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 1)
	marathon := marathon.MarathonerStubForApps(app)

	queue, awaitFunc := testEventHandler(handlerStubs{marathon: marathon})

	body := []byte(`{
	  "appId":"/test/app",
	  "taskId":"test_app.1",
	  "version":123,
	  "alive":false,
	  "eventType":"health_status_changed_event",
	  "timestamp":"2015-12-07T09:33:50.069Z"
	}`)

	// when
	queue <- event{eventType: "health_status_changed_event", timestamp: time.Now(), body: body}
	awaitFunc()

	// then
	assert.False(t, marathon.Interactions())
}

func TestEventHandler_HandleHealthStatusEventReturn202WhenMarathonReturnedError(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)

	queue, awaitFunc := testEventHandler(handlerStubs{marathon: marathon})

	body := []byte(`{
	  "appId":"unknown",
	  "taskId":"unknown.1",
	  "version":"2015-12-07T09:02:48.981Z",
	  "alive":true,
	  "eventType":"health_status_changed_event",
	  "timestamp":"2015-12-07T09:33:50.069Z"
	}`)

	// when
	queue <- event{eventType: "health_status_changed_event", timestamp: time.Now(), body: body}
	awaitFunc()

	// then
	assert.True(t, marathon.Interactions())
}

func TestEventHandler_HandleHealthStatusEventWhenTaskIsNotInMarathon(t *testing.T) {
	t.Parallel()

	// given
	app := ConsulApp("/test/app", 3)
	marathon := marathon.MarathonerStubForApps(app)

	queue, awaitFunc := testEventHandler(handlerStubs{marathon: marathon})

	body := healthStatusChangeEventForTask("unknown.1")

	// when
	queue <- event{eventType: "health_status_changed_event", timestamp: time.Now(), body: body}
	awaitFunc()

	// then
	assert.True(t, marathon.Interactions())
}

type BadReader struct{}

func (r BadReader) Read(p []byte) (int, error) {
	return 0, errors.New("Some error")
}

func healthStatusChangeEventForTask(taskID string) []byte {
	return []byte(`{
	  "appId":"/test/app",
	  "taskId":"` + taskID + `",
	  "version":"2015-12-07T09:02:48.981Z",
	  "alive":true,
	  "eventType":"health_status_changed_event",
	  "timestamp":"2015-12-07T09:33:50.069Z"
	}`)
}
