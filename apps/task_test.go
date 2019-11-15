package apps

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/allegro/marathon-consul/time"
	"github.com/stretchr/testify/assert"
)

func TestParseTask(t *testing.T) {
	t.Parallel()

	testTask := &Task{
		ID:                 "my-app_0-1396592784349",
		Timestamp:          time.Timestamp{},
		AppID:              "/my-app",
		Host:               "slave-1234.acme.org",
		Ports:              []int{31372},
		HealthCheckResults: []HealthCheckResult{{Alive: true}},
	}

	jsonified, err := json.Marshal(testTask)
	assert.Nil(t, err)

	service, err := ParseTask(jsonified)
	assert.Nil(t, err)

	assert.Equal(t, testTask.ID, service.ID)
	assert.Equal(t, testTask.AppID, service.AppID)
	assert.Equal(t, testTask.Host, service.Host)
	assert.Equal(t, testTask.Ports, service.Ports)
	assert.Equal(t, testTask.HealthCheckResults[0].Alive, service.HealthCheckResults[0].Alive)
}

func TestParseTasks(t *testing.T) {
	t.Parallel()

	tasksBlob, _ := ioutil.ReadFile("testdata/tasks.json")

	expectedTasks := []Task{
		{
			ID:                 "test.47de43bd-1a81-11e5-bdb6-e6cb6734eaf8._app.1",
			Timestamp:          time.Timestamp{},
			AppID:              "/test",
			Host:               "192.168.2.114",
			Ports:              []int{31315},
			HealthCheckResults: []HealthCheckResult{{Alive: true}},
		},
		{
			ID:    "test.4453212c-1a81-11e5-bdb6-e6cb6734eaf8._app.1",
			AppID: "/test",
			Host:  "192.168.2.114",
			Ports: []int{31797},
		},
	}

	tasks, err := ParseTasks(tasksBlob)
	assert.Nil(t, err)
	assert.Len(t, tasks, 2)
	assert.Equal(t, expectedTasks, tasks)
}

func TestIsHealthy(t *testing.T) {
	t.Parallel()

	// given
	task := &Task{}

	// when
	task.HealthCheckResults = nil

	// then
	assert.False(t, task.IsHealthy())

	// when
	task.HealthCheckResults = []HealthCheckResult{}

	// then
	assert.False(t, task.IsHealthy())

	// when
	task.HealthCheckResults = []HealthCheckResult{
		{Alive: false},
	}

	// then
	assert.False(t, task.IsHealthy())

	// when
	task.HealthCheckResults = []HealthCheckResult{
		{Alive: true},
		{Alive: false},
	}

	// then
	assert.False(t, task.IsHealthy())

	// when
	task.HealthCheckResults = []HealthCheckResult{
		{Alive: true},
		{Alive: true},
	}

	// then
	assert.True(t, task.IsHealthy())

	// when
	task.State = "TASK_KILLING"

	// then
	assert.False(t, task.IsHealthy())

	// when
	task.State = "TASK_RUNNING"

	// then
	assert.True(t, task.IsHealthy())
}

func TestId_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "id", TaskID("id").String())
}

func TestId_AppId(t *testing.T) {
	t.Parallel()
	id := "pl.allegro_test_app.a7cde60e-0093-11e6-ab55-02aab772a161"
	assert.Equal(t, AppID("/pl.allegro/test/app"), TaskID(id).AppID())
}

func TestId_AppIdForInvalidIdShouldPanic(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		a := TaskID("id").AppID()
		assert.Nil(t, a)
	})
}

func TestFindTaskByIdNotExactMatch(t *testing.T) { // Marathon version 1.9 doesn't match the event app id with the current app id, because it has not the suffix "._app.1"
	t.Parallel()

	tasksBlob, _ := ioutil.ReadFile("testdata/tasks.json")
	tasks, err := ParseTasks(tasksBlob)
	assert.Nil(t, err)

	task := TaskID("test.4453212c-1a81-11e5-bdb6-e6cb6734eaf8")
	_, found := FindTaskByID(task, tasks)
	assert.True(t, found)
}

func TestFindTaskByIdNotFound(t *testing.T) {
	t.Parallel()

	tasksBlob, _ := ioutil.ReadFile("testdata/tasks.json")
	tasks, err := ParseTasks(tasksBlob)
	assert.Nil(t, err)

	task := TaskID("this-task-doesnt-exist")
	_, found := FindTaskByID(task, tasks)
	assert.False(t, found)
}

func TestFindTaskByIdExactMatch(t *testing.T) { // When marathon app id in events is the same as the app id in the task, which is also currently the case.
	t.Parallel()

	tasksBlob, _ := ioutil.ReadFile("testdata/tasks.json")
	tasks, err := ParseTasks(tasksBlob)
	assert.Nil(t, err)

	task := TaskID("test.47de43bd-1a81-11e5-bdb6-e6cb6734eaf8._app.1")
	_, found := FindTaskByID(task, tasks)
	assert.True(t, found)
}

func TestFindTaskByIdExactMatchBeforeMarathonVersionOneDotNine(t *testing.T) { // before marathon v1.9 the app id of an event did match the task app id
	t.Parallel()

	tasksBlob, _ := ioutil.ReadFile("testdata/tasks-before-marathon-1.9.json")
	tasks, err := ParseTasks(tasksBlob)
	assert.Nil(t, err)

	task := TaskID("test.47de43bd-1a81-11e5-bdb6-e6cb6734eaf8")
	_, found := FindTaskByID(task, tasks)
	assert.True(t, found)
}
