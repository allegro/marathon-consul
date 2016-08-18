package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/allegro/marathon-consul/apps"
)

func TestMarathonTaskTAg(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "marathon-task:my-task", MarathonTaskTag(apps.TaskId("my-task")))
}

func TestServiceTaskId(t *testing.T) {
	t.Parallel()
	// given
	service := Service{
		ID: "123",
		Name: "abc",
		RegisteringAgentAddress: "localhost",
		Tags: []string{MarathonTaskTag("my-task")},
	}

	// when
	id, err := service.TaskId()

	// then
	assert.Equal(t, apps.TaskId("my-task"), id)
	assert.NoError(t, err)
}

func TestServiceTaskId_NoMarathonTaskTag(t *testing.T) {
	t.Parallel()
	// given
	service := Service{
		ID: "123",
		Name: "abc",
		RegisteringAgentAddress: "localhost",
		Tags: []string{},
	}

	// when
	_, err := service.TaskId()

	// then
	assert.Error(t, err)
}
