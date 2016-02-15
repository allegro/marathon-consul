package apps

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTask(t *testing.T) {
	t.Parallel()

	testTask := &Task{
		ID:                 "my-app_0-1396592784349",
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

	tasksBlob, _ := ioutil.ReadFile("tasks.json")

	expectedTasks := []*Task{
		{
			ID:                 "test.47de43bd-1a81-11e5-bdb6-e6cb6734eaf8",
			AppID:              "/test",
			Host:               "192.168.2.114",
			Ports:              []int{31315},
			HealthCheckResults: []HealthCheckResult{{Alive: true}},
		},
		{
			ID:    "test.4453212c-1a81-11e5-bdb6-e6cb6734eaf8",
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
}

func TestId_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "id", TaskId("id").String())
}
